package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Student struct defines the structure for student records
type Student struct {
	EnrollmentNumber string `json:"enrollment_number"`
	Name             string `json:"name"`
	Age              int    `json:"age"`
	Class            string `json:"class"`
	Subject          string `json:"subject"`
	IsDeleted        bool   `json:"-"`
}

// In-memory database
var students = make(map[string]Student)
var mu sync.Mutex

// Logger setup
var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	// Create log file
	file, err := os.OpenFile("student-api.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Initialize loggers
	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// POST /student/v1/students - Create a new student
func createStudent(w http.ResponseWriter, r *http.Request) {
	var student Student
	err := json.NewDecoder(r.Body).Decode(&student)
	if err != nil {
		ErrorLogger.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	student.EnrollmentNumber = uuid.New().String()
	mu.Lock()
	students[student.EnrollmentNumber] = student
	mu.Unlock()

	InfoLogger.Printf("Created student: %v", student)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"enrollment_number": student.EnrollmentNumber})
}

// GET /student/v1/students/{studentId} - Get a single student by ID
func getStudent(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["studentId"]

	mu.Lock()
	student, exists := students[id]
	mu.Unlock()

	if !exists || student.IsDeleted {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	InfoLogger.Printf("Retrieved student: %v", student)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

// GET /student/v1/students - Get all students
func getAllStudents(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	var result []Student
	for _, student := range students {
		if !student.IsDeleted {
			result = append(result, student)
		}
	}

	InfoLogger.Printf("Retrieved all students")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DELETE /student/v1/students/{studentId} - Soft delete a student by ID
func deleteStudent(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["studentId"]

	mu.Lock()
	student, exists := students[id]
	if exists {
		student.IsDeleted = true
		students[id] = student
	}
	mu.Unlock()

	if !exists {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	InfoLogger.Printf("Deleted student: %v", student)
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/student/v1/students", createStudent).Methods("POST")
	r.HandleFunc("/student/v1/students", getAllStudents).Methods("GET")
	r.HandleFunc("/student/v1/students/{studentId}", getStudent).Methods("GET")
	r.HandleFunc("/student/v1/students/{studentId}", deleteStudent).Methods("DELETE")

	InfoLogger.Println("Starting server on port 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		ErrorLogger.Fatalf("Failed to start server: %v", err)
	}
}
