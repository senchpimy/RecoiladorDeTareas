package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExtractTasksWithOllama(t *testing.T) {
	mockResponse := OllamaResponse{
		Message: Message{
			Role:    "assistant",
			Content: "- [ ] @{2026-02-20} / TestSubject / Test Task Description",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer ts.Close()

	originalURL := ollamaURL
	ollamaURL = ts.URL
	defer func() { ollamaURL = originalURL }()

	content := "Dummy content"
	filename := "2026-01-13 TestFile.md"
	subject := "TestSubject"

	tasks := extractTasksWithOllama(content, filename, subject)

	if !strings.Contains(tasks, "Test Task Description") {
		t.Errorf("Expected task description not found in response: %s", tasks)
	}
}

func TestAppendTasksToFileModel(t *testing.T) {
	mutex.Lock()
	fileModel = []FileLine{}
	markdownPath = "test_pendientes.md"
	os.WriteFile(markdownPath, []byte(""), 0644)
	mutex.Unlock()

	defer os.Remove(markdownPath)

	tasksText := "- [ ] @{2026-10-10} / GoTest / Task 1\n- [ ] Task 2 without date"

	appendTasksToFileModel(tasksText)

	mutex.RLock()
	defer mutex.RUnlock()

	if len(fileModel) != 2 {
		t.Errorf("Expected 2 tasks in model, got %d", len(fileModel))
	}

	if fileModel[0].Content != "@{2026-10-10} / GoTest / Task 1" {
		t.Errorf("Unexpected content for task 1: %s", fileModel[0].Content)
	}
}

func TestProcessFileIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scanner_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mockResponse := OllamaResponse{
		Message: Message{
			Role:    "assistant",
			Content: "- [ ] @{2026-05-05} / Integration / Task from File",
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer ts.Close()

	originalURL := ollamaURL
	ollamaURL = ts.URL
	defer func() { ollamaURL = originalURL }()

	subjectDir := filepath.Join(tmpDir, "IntegrationSubject")
	os.Mkdir(subjectDir, 0755)

	filename := time.Now().Format("2006-01-02") + " Notes.md"
	filePath := filepath.Join(subjectDir, filename)
	initialContent := "# Notes\nSome notes here."
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	mutex.Lock()
	fileModel = []FileLine{}
	markdownPath = filepath.Join(tmpDir, "pendientes.md")
	os.WriteFile(markdownPath, []byte(""), 0644)
	mutex.Unlock()

	scanAndProcessDirectory(tmpDir)

	processedContentBytes, _ := os.ReadFile(filePath)
	processedContent := string(processedContentBytes)
	if !strings.Contains(processedContent, "procesado_por_ia: true") {
		t.Errorf("File was not marked as processed. Content:\n%s", processedContent)
	}

	mutex.RLock()
	found := false
	for _, line := range fileModel {
		if strings.Contains(line.Content, "Task from File") {
			found = true
			break
		}
	}
	mutex.RUnlock()

	if !found {
		t.Errorf("Task was not added to fileModel")
	}
}
