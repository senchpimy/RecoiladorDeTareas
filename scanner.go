package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

var (
	defaultScanDir string
	ollamaModel    string
	geminiModel    string
	ollamaURL      string
	useGemini      bool
)

const (
	daysToReview   = 7
	metadataHeader = "---\nprocesado_por_ia: true\n---\n\n"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No se pudo cargar el archivo .env, usando variables de entorno del sistema")
	}

	defaultScanDir = os.Getenv("DIRECTORIO_NOTAS")
	ollamaModel = os.Getenv("OLLAMA_MODEL")
	ollamaURL = os.Getenv("OLLAMA_URL")
	geminiModel = os.Getenv("GEMINI_MODEL")
	
	// Si la variable USE_GEMINI es "true", activamos Gemini
	useGemini = os.Getenv("USE_GEMINI") == "true"

	if defaultScanDir == "" {
		log.Println("ADVERTENCIA: DIRECTORIO_NOTAS no definido")
	}
	if ollamaURL == "" {
		// Valor por defecto seguro si no se define, para evitar crashes
		ollamaURL = "http://localhost:11434/api/chat"
	}
}

type OllamaRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaResponse struct {
	Message Message `json:"message"`
}

func scanAndProcessDirectory(scanDir string) {
	if _, err := os.Stat(scanDir); os.IsNotExist(err) {
		log.Printf("Directorio de escaneo no encontrado: %s", scanDir)
		return
	}

	log.Println("Iniciando escaneo de directorio...")
	err := filepath.WalkDir(scanDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			processFile(path)
		}
		return nil
	})
	if err != nil {
		log.Printf("Error al escanear directorio: %v", err)
	}
	log.Println("Escaneo finalizado.")
}

func processFile(path string) {
	filename := filepath.Base(path)
	if !strings.HasSuffix(filename, ".md") {
		return
	}

	dateRegex := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	match := dateRegex.FindStringSubmatch(filename)
	if match == nil {
		return
	}

	fileDate, err := time.Parse("2006-01-02", match[1])
	if err != nil {
		return
	}

	if time.Since(fileDate).Hours() > 24*daysToReview {
		return
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error leyendo archivo %s: %v", path, err)
		return
	}
	content := string(contentBytes)

	if strings.Contains(content, "procesado_por_ia: true") {
		return
	}

	log.Printf("Procesando archivo nuevo: %s", filename)
	subject := filepath.Base(filepath.Dir(path))
	
	// Delegar la extracción a la función agnóstica
	tasks := extractTasks(content, filename, subject)

	if tasks != "None" && tasks != "" {
		lines := strings.Split(tasks, "\n")
		for _, lineStr := range lines {
			trimmed := strings.TrimSpace(lineStr)
			if strings.HasPrefix(trimmed, "- [ ]") {
				// Parse the task line to extract content, checked status, and completed_at
				// This logic is similar to what was in loadFullFileStructure and appendTasksToFileModel
				content := strings.TrimPrefix(trimmed, "- [ ]")
				content = strings.TrimSpace(content)

				// Extract completed_at if present in the format @{YYYY-MM-DD HH:MM:SS}
				var completedAt *time.Time
				if dateIndex := strings.LastIndex(content, " @{"); dateIndex != -1 && strings.HasSuffix(content, "}") {
					dateStr := content[dateIndex+3 : len(content)-1]
					if t, err := time.Parse(TimeFormat, dateStr); err == nil {
						completedAt = &t
					}
					content = strings.TrimSpace(content[:dateIndex])
				}

				p := Pendiente{
					Text:        content,
					Checked:     false, // Newly extracted tasks are unchecked by default
					CompletedAt: completedAt,
				}

				// Use the mutex defined in server.go to protect DB access
				mutex.Lock()
				err = insertTaskIntoDB(p)
				mutex.Unlock()

				if err != nil {
					log.Printf("Error al insertar tarea '%s' en la DB: %v", p.Text, err)
				} else {
					log.Printf("Tarea insertada: %s", p.Text)
				}
			}
		}
		markFileAsProcessed(path, content)
	} else if tasks == "None" {
		log.Printf("No se encontraron tareas en %s. Marcando como procesado.", filename)
		markFileAsProcessed(path, content)
	}
}

// extractTasks decide qué backend usar basado en la variable global useGemini
func extractTasks(content, filename, subject string) string {
	if useGemini {
		return extractTasksWithGemini(content, filename, subject)
	}
	return extractTasksWithOllama(content, filename, subject)
}

func extractTasksWithGemini(content, filename, subject string) string {
	ctx := context.Background()
	// El cliente toma la API KEY de la variable de entorno GEMINI_API_KEY por defecto si config es nil
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Printf("Error creando cliente Gemini: %v", err)
		return ""
	}

	systemPrompt := `Dado el siguiente archivo markdown, extrae una lista de tareas o pendientes que se pueden identificar en el contenido. Si no hay tareas, responde vacio.
si hay tareas, responde con una lista en formato markdown, cada tarea debe empezar con un guión.
No agregues nada más, solo la lista de tareas.
No agregues explicaciones ni introducciones, solo la lista de tareas.
La lista debe ser como la siguiente:
    - [ ] @{ *fecha de entrega en formato YYYY-MM-DD* } / *Materia* / *Descripcion*
Asegúrate de que las fechas de entrega estén en el formato @{YYYY-MM-DD} y si no existe una fecha de entrega, asume que la fecha de entrega es el dia siguiente
Si no puedes encontrar una materia, usa "General" como materia.
Divide la fecha de entrega, la materia y la descripcion con una barra inclinada (/).

Si no hay tareas, responde "None".`

	// Construimos el contexto completo en texto plano para Gemini
	fullPrompt := fmt.Sprintf("%s\n\nEjemplo Usuario:\n%s\n\nEjemplo Asistente:\n%s\n\nEjemplo Usuario 2:\n%s\n\nEjemplo Asistente 2:\n%s\n\nTarea Actual:\nEl nombre de la materia es %s,\nFecha actual: %s\nDia de la semana actual: %s\nNombre del archivo: %s\n```markdown\n%s\n```",
		systemPrompt,
		generateEmptyTasksExample(),
		"None",
		generateTasksExample(),
		"- [ ] @{2025-08-31} / Internet of Things / Construir una cerradura combinacional con 8 entradas y 5 digitos, verificar la contraseña al presionar enter, preparar documentación en PDF (incluyendo circuito, diagrama de bloques, diagrama eléctrico, código fuente y circuito funcionando)",
		subject,
		time.Now().Format("2006-01-02"),
		time.Now().Weekday().String(),
		filename,
		content,
	)

	result, err := client.Models.GenerateContent(
		ctx,
		geminiModel,
		genai.Text(fullPrompt),
		nil,
	)
	if err != nil {
		log.Printf("Error llamando a Gemini API: %v", err)
		return ""
	}

	return result.Text()
}

func extractTasksWithOllama(content, filename, subject string) string {
	prompt := fmt.Sprintf(`
        El nombre de la materia es %s,


        Fecha actual: %s
        Dia de la semana actual: %s
        Nombre del archivo: %s
	%s
	%s
	%s
	`,
		subject,
		time.Now().Format("2006-01-02"),
		time.Now().Weekday().String(),
		filename,
		"```markdown",
		content,
		"```",
	)

	reqBody := OllamaRequest{
		Model:  ollamaModel,
		Stream: false,
		Messages: []Message{
			{Role: "system", Content: `Dado el siguiente archivo markdown, extrae una lista de tareas o pendientes que se pueden identificar en el contenido. Si no hay tareas, responde vacio.
si hay tareas, responde con una lista en formato markdown, cada tarea debe empezar con un guión.
No agregues nada más, solo la lista de tareas.
No agregues explicaciones ni introducciones, solo la lista de tareas.
La lista debe ser como la siguiente:
    - [ ] @{ *fecha de entrega en formato YYYY-MM-DD* } / *Materia* / *Descripcion*
Asegúrate de que las fechas de entrega estén en el formato @{YYYY-MM-DD} y si no existe una fecha de entrega, asume que la fecha de entrega es el dia siguiente
Si no puedes encontrar una materia, usa "General" como materia.
Divide la fecha de entrega, la materia y la descripcion con una barra inclinada (/).

Si no hay tareas, responde "None".`},
			{Role: "user", Content: generateEmptyTasksExample()},
			{Role: "assistant", Content: "None"},
			{Role: "user", Content: generateTasksExample()},
			{Role: "assistant", Content: "- [ ] @{2025-08-31} / Internet of Things / Construir una cerradura combinacional con 8 entradas y 5 digitos, verificar la contraseña al presionar enter, preparar documentación en PDF (incluyendo circuito, diagrama de bloques, diagrama eléctrico, código fuente y circuito funcionando)"},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post(ollamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error conectando con Ollama: %v", err)
		return ""
	}
	defer resp.Body.Close()

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		log.Printf("Error decodificando respuesta de Ollama: %v", err)
		return ""
	}

	return ollamaResp.Message.Content
}



func markFileAsProcessed(path, content string) {
	newContent := metadataHeader + content
	err := os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		log.Printf("Error marcando archivo como procesado %s: %v", path, err)
	} else {
		log.Printf("Archivo marcado como procesado: %s", filepath.Base(path))
	}
}

func generateEmptyTasksExample() string {
	return `
    El nombre de la materia es Sistemas de Informacion,


    Fecha actual: 2023-10-10
    Dia de la semana actual: Lunes
    Nombre del archivo: "Sistemas de Informacion 2023-10-09.md"
    ... (contenido irrelevante) ...
    `
}

func generateTasksExample() string {
	return `
    El nombre de la materia es, Internet of Things,


    Fecha actual: 2023-08-28
    Dia de la semana actual: Jueves
    Nombre del archivo: "Sistemas de Informacion 2023-08-27.md"
    
    Construir una cerradura combinacional...
    `
}