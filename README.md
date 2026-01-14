# TareasGenerador

TareasGenerador is a powerful system designed to automate the extraction and management of tasks from your Markdown notes. Utilizing Artificial Intelligence (AI) from either Ollama or Google Gemini, it continuously scans a designated directory, identifies actionable items within your notes, stores them in a local SQLite database, and provides a RESTful API for easy access and management.

### Prerequisites

*   **Go:** Version 1.22 or higher.
*   **Ollama (Optional):** If you plan to use Ollama for AI processing, ensure it's installed and running.
*   **Google Cloud Account (Optional):** If you plan to use Google Gemini, ensure you have a Google Cloud account with the Generative AI API enabled and a valid API key.

### Cloning the Repository

```bash
git clone https://github.com/senchpimy/TareasGenerador.git
cd TareasGenerador
```

### Setting up Environment Variables

Create a `.env` file in the root directory of the project. This file will hold your configuration.

```dotenv
# Path to the directory containing your Markdown notes.
# Example: /home/user/notes
DIRECTORIO_NOTAS="/path/to/your/markdown/notes"

# --- AI Configuration ---
# Set to "true" to use Google Gemini, otherwise Ollama will be used.
USE_GEMINI="false" 

# Configuration for Ollama (if USE_GEMINI is "false")
OLLAMA_MODEL="llama2" # The Ollama model to use (e.g., llama2, mistral)
OLLAMA_URL="http://localhost:11434/api/chat" # URL for your Ollama instance

# Configuration for Google Gemini (if USE_GEMINI is "true")
# GEMINI_API_KEY is usually picked up automatically by the Google GenAI client
# from your environment if set globally. If not, you might need to
# provide it directly depending on your setup.
GEMINI_MODEL="gemini-pro" # The Gemini model to use (e.g., gemini-pro)
```

**Note:** If `GEMINI_API_KEY` is not set globally in your environment, you might need to configure it in your application code or ensure it's picked up by the `genai` client library.

### Running the Application

1.  **Install Go Dependencies:**
    ```bash
    go mod tidy
    ```
2.  **Build and Run:**
    ```bash
    go run server.go scanner.go
    ```
    The server will start on `http://localhost:8080`.

    Alternatively, you can build an executable:
    ```bash
    go build -o tareasgenerador .
    ./tareasgenerador
    ```

### SQLite Database Location

The SQLite database `tasks.db` will be created in your user's data directory:
`~/.local/share/tareasgenerador/tasks.db`

## API Endpoints

The application exposes the following RESTful API endpoints:

### 1. Get All Tasks

Retrieves a list of all tasks currently stored in the database.

*   **URL:** `/pendientes`
*   **Method:** `GET`
*   **Response (JSON Array):**
    ```json
    [
      {
        "id": 1,
        "text": "Call John about project X",
        "checked": false,
        "completed_at": null
      },
      {
        "id": 2,
        "text": "Review pull request #123",
        "checked": true,
        "completed_at": "2024-01-14 10:30:00"
      }
    ]
    ```

### 2. Update Task Status

Updates the `checked` status of a specific task. If `checked` is set to `true`, `completed_at` will be set to the current timestamp. If set to `false`, `completed_at` will be `null`.

*   **URL:** `/update`
*   **Method:** `POST`
*   **Request Body (JSON):**
    ```json
    {
      "id": 1,
      "checked": true
    }
    ```
*   **Response (JSON):**
    ```json
    {
      "status": "ok"
    }
    ```

## Python Scripts (Experimental/Alternative)

The `python_ver` directory contains experimental or alternative Python scripts that offer similar note processing capabilities, primarily focusing on summarization and console reporting. These are standalone and do not interact with the Go application's database or API.

*   `main.py`: Scans directories for Markdown notes, processes them, and generates a console report.
*   `resumidor.py`: Contains the logic for summarizing notes, likely using an LLM.
*   `test_ollama.py`: Used for testing Ollama integration within the Python environment.
