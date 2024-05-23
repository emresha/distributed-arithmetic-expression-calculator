package main

import (
	"distributed-calculator/api/handler"
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", handler.AddCalculation)
	mux.HandleFunc("/api/v1/expressions", handler.HandleAllExpressions)
	mux.HandleFunc("/api/v1/expressions/", handler.HandleAllExpressions)
	mux.HandleFunc("/internal/task", handler.HandleCalculations)
	fmt.Println("Running server...")
	http.ListenAndServe("localhost:8080", mux)
}
