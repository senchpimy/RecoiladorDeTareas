package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"database/sql"
	"path/filepath" // Add this import
	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library

)

const TimeFormat = "2006-01-02 15:04:05"

type Pendiente struct {
	ID          int        `json:"id"`
	Text        string     `json:"text"`
	Checked     bool       `json:"checked"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

var (
	db    *sql.DB
	mutex = &sync.RWMutex{}
)





func insertTaskIntoDB(p Pendiente) error {
	stmt, err := db.Prepare("INSERT INTO tasks(text, checked, completed_at) VALUES(?, ?, ?)")
	if err != nil {
		return fmt.Errorf("error preparing insert statement: %w", err)
	}
	defer stmt.Close()

	var completedAtStr sql.NullString
	if p.CompletedAt != nil {
		completedAtStr.String = p.CompletedAt.Format(TimeFormat)
		completedAtStr.Valid = true
	} else {
		completedAtStr.Valid = false
	}

	_, err = stmt.Exec(p.Text, p.Checked, completedAtStr)
	if err != nil {
		return fmt.Errorf("error executing insert statement: %w", err)
	}
	return nil
}

func getTasksFromDB() ([]Pendiente, error) {
	rows, err := db.Query("SELECT id, text, checked, completed_at FROM tasks ORDER BY id DESC")
	if err != nil {
		return nil, fmt.Errorf("error querying tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Pendiente
	for rows.Next() {
		var p Pendiente
		var completedAtStr sql.NullString
		err := rows.Scan(&p.ID, &p.Text, &p.Checked, &completedAtStr)
		if err != nil {
			return nil, fmt.Errorf("error scanning task row: %w", err)
		}

		if completedAtStr.Valid {
			t, err := time.Parse(TimeFormat, completedAtStr.String)
			if err != nil {
				log.Printf("Error parsing completed_at time: %v", err)
				// Continue with nil if parsing fails to avoid blocking other tasks
			} else {
				p.CompletedAt = &t
			}
		}
		tasks = append(tasks, p)
	}

	return tasks, nil
}

func getPendientesHandler(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	defer mutex.RUnlock()

	tasks, err := getTasksFromDB()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al obtener las tareas: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}



type UpdateRequest struct {
	ID      int  `json:"id"`
	Checked bool `json:"checked"`
}

func updateTaskInDB(id int, checked bool) error {
	var stmt *sql.Stmt
	var err error

	if checked {
		// Update checked status and set completed_at to now
		stmt, err = db.Prepare("UPDATE tasks SET checked = ?, completed_at = ? WHERE id = ?")
		if err != nil {
			return fmt.Errorf("error preparing update statement (checked): %w", err)
		}
		defer stmt.Close()

		now := time.Now()
		_, err = stmt.Exec(checked, now.Format(TimeFormat), id)
	} else {
		// Update checked status and set completed_at to NULL
		stmt, err = db.Prepare("UPDATE tasks SET checked = ?, completed_at = NULL WHERE id = ?")
		if err != nil {
			return fmt.Errorf("error preparing update statement (unchecked): %w", err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(checked, id)
	}

	if err != nil {
		return fmt.Errorf("error executing update statement: %w", err)
	}
	return nil
}

func updatePendienteHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	if err := updateTaskInDB(req.ID, req.Checked); err != nil {
		log.Printf("¡ATENCIÓN! Error al actualizar tarea en la DB: %v", err)
		http.Error(w, "Error interno al actualizar la tarea", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// migrateMarkdownToSQLite reads pendientes.md and populates the SQLite database.
func migrateMarkdownToSQLite(markdownFilePath string) error {
	file, err := os.Open(markdownFilePath)
	if os.IsNotExist(err) {
		log.Printf("Markdown file %s does not exist, skipping migration.", markdownFilePath)
		return nil // No file to migrate, not an error
	}
	if err != nil {
		return fmt.Errorf("error opening markdown file for migration: %w", err)
	}
	defer file.Close()

	// Local FileLine struct for migration purposes
	type FileLine struct {
		IsTask      bool
		Content     string
		Indentation string
		Checked     bool
		CompletedAt *time.Time
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		taskMarkerIndex := strings.Index(line, "- [")

		if taskMarkerIndex != -1 {
			var fl FileLine
			fl.IsTask = true
			fl.Indentation = line[:taskMarkerIndex]
			taskContent := strings.TrimSpace(line[taskMarkerIndex:])

			if strings.HasPrefix(taskContent, "- [x]") {
				fl.Checked = true
				fullText := strings.TrimSpace(strings.TrimPrefix(taskContent, "- [x]"))
				if dateIndex := strings.LastIndex(fullText, " @{"); dateIndex != -1 && strings.HasSuffix(fullText, "}") {
					fl.Content = strings.TrimSpace(fullText[:dateIndex])
					dateStr := fullText[dateIndex+3 : len(fullText)-1]
					if t, err := time.Parse(TimeFormat, dateStr); err == nil {
						fl.CompletedAt = &t
					}
				} else {
					fl.Content = fullText
				}
			} else { // Handle [ ] and [-] as unchecked
				fl.Checked = false
				contentWithoutMarker := strings.TrimPrefix(taskContent, "- [ ]")
				contentWithoutMarker = strings.TrimPrefix(contentWithoutMarker, "- [-]")
				fl.Content = strings.TrimSpace(contentWithoutMarker)
			}

			// Insert into DB
			task := Pendiente{
				Text:        fl.Content,
				Checked:     fl.Checked,
				CompletedAt: fl.CompletedAt,
			}
			if err := insertTaskIntoDB(task); err != nil {
				log.Printf("Error inserting migrated task '%s': %v", task.Text, err)
			}
		}
	}
	return scanner.Err()
}

func main() {
	// Initialize SQLite
	var err error
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory: %v", err)
	}
	dataDir := filepath.Join(homeDir, ".local", "share", "tareasgenerador")
	dbPath := filepath.Join(dataDir, "tasks.db")

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Error creating data directory: %v", err)
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		text TEXT NOT NULL,
		checked BOOLEAN NOT NULL DEFAULT FALSE,
		completed_at TEXT
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("Error creating tasks table: %v", err)
	}

	// Check if the database is empty before migrating
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
	if err != nil {
		log.Fatalf("Error checking task count: %v", err)
	}
	if count == 0 {
		log.Println("Database is empty, attempting to migrate from pendientes.md...")
		if err := migrateMarkdownToSQLite("./pendientes.md"); err != nil {
			log.Fatalf("Error during markdown migration: %v", err)
		}
		log.Println("Migration from pendientes.md completed successfully (if file existed).")
	} else {
		log.Println("Database already contains tasks, skipping migration from pendientes.md.")
	}

	log.Println("Servidor de horarios iniciado. Sirviendo en http://localhost:8080")

	// Iniciar escáner de carpetas en segundo plano
	go func() {
		log.Println("Iniciando escáner de carpetas inicial...")
		scanAndProcessDirectory(defaultScanDir)
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			log.Println("Ejecutando escaneo periódico...")
			scanAndProcessDirectory(defaultScanDir)
		}
	}()

	corsHandler := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				return
			}
			h(w, r)
		}
	}

	http.HandleFunc("/pendientes", corsHandler(getPendientesHandler))
	http.HandleFunc("/update", corsHandler(updatePendienteHandler))


	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("No se pudo iniciar el servidor: %v", err)
	}
}
