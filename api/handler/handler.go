/*
Package handler is made for handling different endpoints.
It defines 2 structs: Task and Calculation.
Task is a type of structure that defines a task given by user, which consists of different Calculations that might be done concurrently.
Calculation is a type of structure that represents a single calculation using Reverse Polisn Notation.
*/

package handler

import (
	"distributed-calculator/internal/logic"
	"distributed-calculator/internal/service"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var Tasks map[int]service.Task

var Calculations chan service.Calculation

func AddCalculation(w http.ResponseWriter, r *http.Request) {
	NewTask := service.Task{}
	if r.Method == http.MethodPost && r.Header["Content-Type"][0] == "application/json" {
		body, err := io.ReadAll(r.Body)
		// handle read error
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// handle json error
		if err = json.Unmarshal(body, &NewTask); err != nil {
			fmt.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// if expression is incorrect, then throw error
		if err = calculate.ValidateInfixExpression(NewTask.Expression); err != nil {
			http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
			return
		}

		// if task already exists, then throw an error because every task must have a unique id!!
		if _, ok := Tasks[NewTask.Id]; ok {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		// keeping all tasks in RAM for now.
		Tasks[NewTask.Id] = NewTask

		newRPN, err := calculate.InfixToRPN(NewTask.Expression)

		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		calculate.RPNtoSeparateCalculations(newRPN, NewTask.Id, Calculations)
		
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "{}")

	} else {
		http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
		return
	}
}

func HandleCalculations(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		select {
		case calculation := <-Calculations: // if there are any calculations available.
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
		// handle read error.
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// handle json error.
		if err = json.Unmarshal(body, &FinishedCalculation); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// just in case some strange magic happened??
		if FinishedCalculation.Status != "Finished" {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// change linked task's expression accordingly
		// and check if task successfully calculated.
		LinkedTask := Tasks[FinishedCalculation.Task_id]
		LinkedTask.Expression = strings.ReplaceAll(LinkedTask.Expression, FinishedCalculation.RPN_string, fmt.Sprintf("%.6f", FinishedCalculation.Result))
		fmt.Println(LinkedTask.Expression)

		// if calculation finished, we change the task status.
		if calculate.IsFloat(LinkedTask.Expression) {
			LinkedTask.Status = "Finished"
			LinkedTask.Result = FinishedCalculation.Result
			Tasks[FinishedCalculation.Task_id] = LinkedTask
		}

		fmt.Fprintf(w, "{}")

	} else {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

}

func HandleAllExpressions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/expressions/")

	// split the path to get the ID
	parts := strings.Split(path, "/")
	id := parts[len(parts)-1]
	// check if ID is present and if no ID, show all expressions
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
