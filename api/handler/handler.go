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

// This slice has calculations waiting to be sent to The Agent.
var Calculations = []service.Calculation{}
// This slice has calculations that are currently being calculated by The Agent.
var BeingCalculated = []service.Calculation{}

func AddTask(w http.ResponseWriter, r *http.Request) {
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

		// If expression is just a number, then it is not an expression.
		if calculate.IsFloat(NewTask.Expression) {
			http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
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

		go calculate.RPNtoSeparateCalculations(newRPN, NewTask.Id, &Calculations, BeingCalculated)


	} else {
		http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
		return
	}
}

func HandleCalculations(w http.ResponseWriter, r *http.Request) {
	// If request method is get, then The Agent is asking for a Calculation to calculate.
	if r.Method == http.MethodGet {
		// If there are calculations waiting to be calculated, hand them out to The Agent. 
		if len(Calculations) > 0 {
			calculation := Calculations[0]

			task_json, err := json.Marshal(calculation)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			// Delete the handed Calculation from waiting list...
			Calculations = Calculations[1:]
			// ... and add it to the BeingCalculated slice!
			BeingCalculated = append(BeingCalculated, calculation)

			w.WriteHeader(http.StatusOK)
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, string(task_json))
			return
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
	} else if r.Method == http.MethodPost && r.Header["Content-Type"][0] == "application/json" {
		// ^^^^^ If the request method is POST, then The Agent has calculated a Calculation
		// and is sending it back.
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
		service.DeleteCalculationFromSlice(FinishedCalculation, &BeingCalculated)
		LinkedTask := Tasks[FinishedCalculation.Task_id]
		if LinkedTask.Status == "Finished" {
			return
		}

		// This is a lot of log messages...
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
			go calculate.RPNtoSeparateCalculations(LinkedExpressionRPN, LinkedTask.Id, &Calculations, BeingCalculated)
		}


	} else {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

}

func HandleAllExpressions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Check if ID is present and if no ID, show all expressions
	if id == "" {
		// The code below shows all expressions.
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
		// If ID is not empty and it is a number, then try to get the Task with the required ID.
		searchedTaskId, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	
		searchedTask, ok := Tasks[searchedTaskId]
		// If not found, then send 404.
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
