/*
Package handler is made for handling different endpoints.
It defines 2 structs: Task and Calculation.
Task is a type of structure that defines a task given by user, which consists of different Calculations that might be done concurrently.
Calculation is a type of structure that represents a single calculation using Reverse Polish Notation.
*/

package handler

import (
	"distributed-calculator/internal/logic"
	"distributed-calculator/internal/service"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var Tasks = make(map[int]service.Task)

var Calculations = make(chan service.Calculation)

func AddCalculation(w http.ResponseWriter, r *http.Request) {
	NewTask := service.Task{}
	if r.Method == http.MethodPost && r.Header["Content-Type"][0] == "application/json" {
		body, err := io.ReadAll(r.Body)
		// Handle read error
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Handle json error
		if err = json.Unmarshal(body, &NewTask); err != nil {
			fmt.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If expression is incorrect, then throw error
		if err = calculate.ValidateInfixExpression(NewTask.Expression); err != nil {
			log.Printf("%v\n", err)
			http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
			return
		}

		// If task already exists, then throw an error because every task must have a unique id!!
		if _, ok := Tasks[NewTask.Id]; ok {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		// Keeping all tasks in RAM for now.
		NewTask.Status = "In Process"
		NewTask.Original_Expression = NewTask.Expression
		Tasks[NewTask.Id] = NewTask

		newRPN, err := calculate.InfixToRPN(NewTask.Expression)
		log.Println(newRPN)

		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "{}")

		go calculate.RPNtoSeparateCalculations(newRPN, NewTask.Id, Calculations)


	} else {
		http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
		return
	}
}

func HandleCalculations(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		select {
		case calculation := <-Calculations: // If there are any calculations available
			task_json, err := json.Marshal(calculation)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, string(task_json))
			return
		default:
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
	} else if r.Method == http.MethodPost && r.Header["Content-Type"][0] == "application/json" {
		FinishedCalculation := service.Calculation{}
		body, err := io.ReadAll(r.Body)
		// Handle read error.
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Handle json error.
		if err = json.Unmarshal(body, &FinishedCalculation); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Just in case some strange magic happened??
		if FinishedCalculation.Status != "Finished" {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "{}")
		
		// Change linked task's expression accordingly
		// And check if task successfully calculated.
		LinkedTask := Tasks[FinishedCalculation.Task_id]
		if LinkedTask.Status == "Finished" {
			return
		}
		log.Printf("Finished calculation: %s, Result: %d\n", FinishedCalculation.RPN_string, FinishedCalculation.Result)
		log.Printf("Linked Task Expression infix: %s\n", LinkedTask.Expression)
		LinkedExpressionRPN, _ := calculate.InfixToRPN(LinkedTask.Expression)
		log.Printf("Linked Task Expression RPN: %s\n", LinkedExpressionRPN)
		LinkedExpressionRPN = strings.ReplaceAll(LinkedExpressionRPN, FinishedCalculation.RPN_string, fmt.Sprintf("%d", FinishedCalculation.Result))
		log.Printf("Linked Task Expression New RPN: %s\n", LinkedExpressionRPN)
		LinkedExpressionInfix, _ := calculate.RPNtoInfix(LinkedExpressionRPN)
		log.Printf("Linked Task Expression New Infix: %s\n", LinkedExpressionInfix)
		LinkedTask.Expression = LinkedExpressionInfix
		Tasks[FinishedCalculation.Task_id] = LinkedTask

		// If calculation finished, change the task status.
		if calculate.IsFloat(LinkedTask.Expression) {
			res, _ := strconv.Atoi(LinkedTask.Expression)
			LinkedTask.Status = "Finished"
			LinkedTask.Result = res
			Tasks[FinishedCalculation.Task_id] = LinkedTask
			log.Printf("FINISHED CALCULATING RESULT IS %d\n", LinkedTask.Result)
		} else {
			go calculate.RPNtoSeparateCalculations(LinkedExpressionRPN, LinkedTask.Id, Calculations)
		}


	} else {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

}

func HandleAllExpressions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/expressions/")

	// Split the path to get the ID; TODO: use new Go 1.22.2 router instead of this terrible code.
	parts := strings.Split(path, "/")
	id := parts[len(parts)-1]
	// Check if ID is present and if no ID, show all expressions
	if id == "expressions" {
		if r.Method == http.MethodGet {
			all_expressions := []service.Task{}
			for _, value := range Tasks {
				all_expressions = append(all_expressions, value)
			}

			expressions, err := json.Marshal(all_expressions)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, string(expressions))
			return
		} else {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	} else if calculate.IsFloat(id) {

		searchedTaskId, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	
		searchedTask, ok := Tasks[searchedTaskId]
		if !ok {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
	
		searchedTaskJson, err := json.Marshal(searchedTask)
	
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(searchedTaskJson))
	} else {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}

}
