package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v2"
)

type Config struct {
	AWS struct {
		S3Bucket string `yaml:"s3_bucket"`
	} `yaml:"aws"`
}

// Quiz represents the structure of the quiz with a unique ID
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
	ID    string `json:"id"`
	Title string `json:"title"`
}

var config Config
var s3Svc *s3.S3

func init() {
	// Load config.yaml
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("Failed to parse config file: %v\n", err)
		os.Exit(1)
	}

	// Initialize S3 client
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"), // Replace with your region
	})
	if err != nil {
		fmt.Printf("Failed to create AWS session: %v\n", err)
		os.Exit(1)
	}

	s3Svc = s3.New(sess)
}

// saveQuizToS3 saves the quiz to the S3 bucket specified in the config
func saveQuizToS3(quiz Quiz) error {
	quizData, err := yaml.Marshal(quiz)
	if err != nil {
		return fmt.Errorf("failed to marshal quiz: %v", err)
	}

	// Generate a unique file name
	fileName := fmt.Sprintf("quiz-%s.yaml", quiz.ID)

	_, err = s3Svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(config.AWS.S3Bucket),
		Key:    aws.String(fileName),
		Body:   bytes.NewReader(quizData),
	})
	if err != nil {
		return fmt.Errorf("failed to upload quiz to S3: %v", err)
	}

	return nil
}

// getQuizFromS3 retrieves a quiz from the S3 bucket
func getQuizFromS3(quizID string) (*Quiz, error) {
	fileName := fmt.Sprintf("%s.yaml", quizID)
	resp, err := s3Svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(config.AWS.S3Bucket),
		Key:    aws.String(fileName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve quiz from S3: %v", err)
	}
	defer resp.Body.Close()

	var quiz Quiz
	err = yaml.NewDecoder(resp.Body).Decode(&quiz)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quiz file: %v", err)
	}

	return &quiz, nil
}

// listQuizzesFromS3 lists all quizzes in the S3 bucket
func listQuizzesFromS3() ([]QuizSummary, error) {
	resp, err := s3Svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.AWS.S3Bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list quizzes from S3: %v", err)
	}

	var summaries []QuizSummary
	for _, item := range resp.Contents {
		quizID := filepath.Base(*item.Key)
		quizID = quizID[:len(quizID)-len(filepath.Ext(quizID))] // remove .yaml extension
		quiz, err := getQuizFromS3(quizID)
		if err != nil {
			return nil, err
		}

		summaries = append(summaries, QuizSummary{ID: quiz.ID, Title: quiz.Title})
	}

	return summaries, nil
}

// deleteQuizFromS3 deletes a quiz from the S3 bucket
func deleteQuizFromS3(quizID string) error {
	fileName := fmt.Sprintf("%s.yaml", quizID)

	_, err := s3Svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(config.AWS.S3Bucket),
		Key:    aws.String(fileName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete quiz from S3: %v", err)
	}

	return nil
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
			fmt.Printf("Failed to parse quiz data: %v", err)
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
		err = saveQuizToS3(quiz)
		if err != nil {
			fmt.Printf("Failed to save quiz: %v", err)
			http.Error(w, fmt.Sprintf("Failed to save quiz: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Quiz saved successfully"))
	}
}

func listQuizzesHandler(w http.ResponseWriter, r *http.Request) {
	summaries, err := listQuizzesFromS3()
	if err != nil {
		fmt.Printf("Failed to list quizzes: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list quizzes: %v", err), http.StatusInternalServerError)
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

	t := template.Must(template.New("quizList").Parse(tmpl))
	err = t.Execute(w, summaries)
	if err != nil {
		fmt.Printf("Failed to render template: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// Handle the GET request to return quiz list in JSON format
func quizListHandler(w http.ResponseWriter, r *http.Request) {
	summaries, err := listQuizzesFromS3()
	if err != nil {
		fmt.Printf("Failed to list quizzes: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list quizzes: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

// Handle the GET request to show a specific quiz
func quizHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		deleteQuizHandler(w, r)
		return
	}
	quizID := filepath.Base(r.URL.Path)

	quiz, err := getQuizFromS3("quiz-" + quizID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Quiz not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	data, _ := yaml.Marshal(quiz)
	w.Write(data)
}

// Handle the DELETE request to delete a specific quiz
func deleteQuizHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	quizID := filepath.Base(r.URL.Path)
	err := deleteQuizFromS3("quiz-" + quizID)
	if err != nil {
		fmt.Printf("Failed to delete quiz: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete quiz: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Quiz deleted successfully"))
}

func main() {
	// Serve static files (HTML, JS, etc.)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Route for the quiz creation page
	http.HandleFunc("/create", createHandler)

	// Route to list all quizzes with delete option
	http.HandleFunc("/quizzes", quizListHandler)

	// Route to return quiz list in JSON format
	http.HandleFunc("/quizlist", listQuizzesHandler)

	// Route to show a specific quiz
	http.HandleFunc("/quiz/", quizHandler)

	fmt.Println("Server started at http://localhost:8081")
	http.ListenAndServe(":8081", nil)
}
