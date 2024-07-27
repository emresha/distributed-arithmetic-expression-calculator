package main

import (
	"distributed-calculator/api/handler"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func main() {

	mux := http.NewServeMux()
	static := filepath.Join("..", "..")
	baseDir, _ := os.Getwd()
	fmt.Println(baseDir)
	fmt.Println(static)
	fs := http.FileServer(http.Dir(static))
	mux.HandleFunc("/", handler.TaskPage)
	mux.Handle("/static/", fs)
	mux.HandleFunc("/api/v1/calculate", handler.AddTask)
	mux.HandleFunc("/api/v1/expressions", handler.HandleAllExpressions)
	mux.HandleFunc("/api/v1/expressions/{id}", handler.HandleAllExpressions)
	mux.HandleFunc("/internal/task", handler.HandleCalculations)
	mux.HandleFunc("/api/v1/register", handler.HandleRegistration)
	mux.HandleFunc("/api/v1/login", handler.HandleLogin)
	fmt.Println("Running server...")
	http.ListenAndServe("localhost:8080", mux)
}
