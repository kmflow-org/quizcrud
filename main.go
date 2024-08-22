package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

type Quiz struct {
	ID        string     `yaml:"id"`
	Title     string     `yaml:"title"`
	Questions []Question `yaml:"questions"`
}

// Question represents a single question in the quiz with a unique ID
type Question struct {
	ID             int      `yaml:"id"`
	Question       string   `yaml:"text"`
	CodeSnippet    string   `yaml:"code,omitempty"`
	Type           string   `yaml:"type"`
	Choices        []string `yaml:"options"`
	CorrectAnswers []int    `yaml:"answers"`
}

type QuizSummary struct {
	ID    string
	Title string
}

// Handle the GET request to serve the HTML page
func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("static/index.html")
		if err != nil {
			http.Error(w, "Could not load template", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	} else if r.Method == http.MethodPost {
		var quiz Quiz

		err := yaml.NewDecoder(r.Body).Decode(&quiz)
		if err != nil {
			http.Error(w, "Failed to parse quiz data", http.StatusBadRequest)
			return
		}

		// Generate a unique ID for the quiz
		quiz.ID = fmt.Sprintf("%d", time.Now().UnixNano())

		// Assign IDs to each question starting from 1
		for i := range quiz.Questions {
			quiz.Questions[i].ID = i + 1
		}

		// Ensure the quizzes directory exists
		err = os.MkdirAll("quizzes", os.ModePerm)
		if err != nil {
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			return
		}

		// Create a file name with a timestamp
		fileName := fmt.Sprintf("quiz-%s.yaml", quiz.ID)
		filePath := filepath.Join("quizzes", fileName)

		// Save the quiz data as a YAML file
		file, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to save quiz", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		err = yaml.NewEncoder(file).Encode(quiz)
		if err != nil {
			http.Error(w, "Failed to write quiz data", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Quiz saved successfully"))
	}
}

func listQuizzesHandler(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir("quizzes")
	if err != nil {
		http.Error(w, "Failed to read quizzes directory", http.StatusInternalServerError)
		return
	}

	tmpl := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Quizzes</title>
			<style>
				.delete-icon {
					cursor: pointer;
					color: red;
					margin-left: 10px;
				}
			</style>
			<script>
				function deleteQuiz(quizId) {
					if (confirm('Are you sure you want to delete this quiz?')) {
						fetch('/quiz/' + quizId, {
							method: 'DELETE'
						}).then(response => {
							if (response.ok) {
								alert('Quiz deleted successfully');
								window.location.reload();
							} else {
								alert('Failed to delete quiz');
							}
						});
					}
				}
			</script>
		</head>
		<body>
			<h1>Saved Quizzes</h1>
			<ul>
				{{range .}}
					<li>
						<a href="/quiz/{{.ID}}">{{.Title}}</a>
						<span class="delete-icon" onclick="deleteQuiz('{{.ID}}')">🗑️</span>
					</li>
				{{end}}
			</ul>
		</body>
		</html>`

	var summaries []QuizSummary

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yaml" {
			data, err := os.ReadFile(filepath.Join("quizzes", file.Name()))
			if err != nil {
				http.Error(w, "Failed to read quiz file", http.StatusInternalServerError)
				return
			}

			var quiz Quiz
			err = yaml.Unmarshal(data, &quiz)
			if err != nil {
				http.Error(w, "Failed to parse quiz file", http.StatusInternalServerError)
				return
			}

			summaries = append(summaries, QuizSummary{ID: quiz.ID, Title: quiz.Title})
		}
	}

	t := template.Must(template.New("quizList").Parse(tmpl))
	err = t.Execute(w, summaries)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// Handle the GET request to show a specific quiz
func showQuizHandler(w http.ResponseWriter, r *http.Request) {
	quizID := filepath.Base(r.URL.Path)
	filePath := filepath.Join("quizzes", fmt.Sprintf("quiz-%s.yaml", quizID))

	if r.Method == http.MethodDelete {
		err := os.Remove(filePath)
		if err != nil {
			http.Error(w, "Failed to delete quiz", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Quiz deleted successfully"))

	} else if r.Method == http.MethodGet {
		data, err := os.ReadFile(filePath)
		if err != nil {
			http.Error(w, "Quiz not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write(data)
	}

}

// Handle the GET request to return quiz list in JSON format
func quizListHandler(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir("quizzes")
	if err != nil {
		http.Error(w, "Failed to read quizzes directory", http.StatusInternalServerError)
		return
	}

	var summaries []QuizSummary

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yaml" {
			data, err := os.ReadFile(filepath.Join("quizzes", file.Name()))
			if err != nil {
				http.Error(w, "Failed to read quiz file", http.StatusInternalServerError)
				return
			}

			var quiz Quiz
			err = yaml.Unmarshal(data, &quiz)
			if err != nil {
				http.Error(w, "Failed to parse quiz file", http.StatusInternalServerError)
				return
			}

			summaries = append(summaries, QuizSummary{ID: quiz.ID, Title: quiz.Title})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

func main() {
	// Serve static files (HTML, JS, etc.)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Route for the quiz creation page
	http.HandleFunc("/create", createHandler)

	// Route to list all quizzes
	http.HandleFunc("/quizzes", listQuizzesHandler)

	// Route to return quiz list in JSON format
	http.HandleFunc("/quizlist", quizListHandler)

	// Route to show a specific quiz
	http.HandleFunc("/quiz/", showQuizHandler)

	fmt.Println("Server started at http://localhost:8081")
	http.ListenAndServe(":8081", nil)
}
