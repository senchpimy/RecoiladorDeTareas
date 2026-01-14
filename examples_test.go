package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

type ExampleScenario struct {
	Name             string
	Filename         string
	Subject          string
	ContentGenerator func() string
	ExpectedContains []string
}

func TestLLMExamplesReal(t *testing.T) {
	today := time.Now()
	nextFriday := today.AddDate(0, 0, (12-int(today.Weekday()))%7)
	if nextFriday.Before(today) {
		nextFriday = nextFriday.AddDate(0, 0, 7)
	}

	scenarios := []ExampleScenario{
		{
			Name:     "Redes (Contexto de clase)",
			Filename: fmt.Sprintf("%s Redes.md", today.Format("2006-01-02")),
			Subject:  "Redes",
			ContentGenerator: func() string {
				return fmt.Sprintf(`
# Clase de Redes
%s

El profesor mencionó que debemos terminar la configuración del router.
- [ ] Configurar VLAN 10
- [ ] Configurar VLAN 20

Tarea para el viernes:
Investigar sobre OSPF y hacer un resumen de 1 cuartilla.
`, today.Format("2006-01-02"))
			},
			ExpectedContains: []string{
				"Redes",
				"Configurar VLAN 10",
				"Investigar sobre OSPF",
			},
		},
		{
			Name:     "Base de Datos (Tareas formales)",
			Filename: fmt.Sprintf("%s BasesDeDatos.md", today.Format("2006-01-02")),
			Subject:  "BasesDeDatos",
			ContentGenerator: func() string {
				return `
Revisión de Normalización.

Para la próxima clase:
1. Traer el diagrama ER de la tienda.
2. Escribir las sentencias SQL para crear las tablas de Usuarios y Productos.
`
			},
			ExpectedContains: []string{
				"BasesDeDatos",
				"diagrama ER",
				"SQL",
			},
		},
		{
			Name:     "General (Lista simple)",
			Filename: fmt.Sprintf("%s NotasRapidas.md", today.Format("2006-01-02")),
			Subject:  "General",
			ContentGenerator: func() string {
				return `
Ir al supermercado:
- Leche
- Huevos

Pagar el recibo de luz antes del 20.
`
			},
			ExpectedContains: []string{
				"General",
				"supermercado",
				"recibo de luz",
			},
		},
	}

	backendName := "Ollama"
	targetModel := ollamaModel
	if useGemini {
		backendName = "Gemini"
		targetModel = geminiModel
	}
	fmt.Printf("Probando contra %s (%s)...\n", backendName, targetModel)

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			content := sc.ContentGenerator()

			start := time.Now()
			result := extractTasks(content, sc.Filename, sc.Subject)
			duration := time.Since(start)

			t.Logf("--- Escenario: %s ---", sc.Name)
			t.Logf("Tiempo de respuesta: %v", duration)
			t.Logf("Respuesta cruda:\n%s\n", result)

			if result == "" || result == "None" {
				t.Errorf("%s devolvió una respuesta vacía o 'None'.", backendName)
				return
			}

			for _, substr := range sc.ExpectedContains {
				if !strings.Contains(result, substr) {
					t.Errorf("Se esperaba encontrar '%s' en la respuesta, pero no estaba.\nRespuesta: %s", substr, result)
				}
			}

			if !strings.Contains(result, "- [ ] @{") {
				t.Errorf("La respuesta no parece tener el formato correcto '- [ ] @{...}'.\nRespuesta: %s", result)
			}
		})
	}
}
