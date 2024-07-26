package handler

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	calculate "distributed-calculator/internal/logic"
	"distributed-calculator/internal/service"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	tasksMutex           sync.Mutex
	calculationsMutex    sync.Mutex
	beingCalculatedMutex sync.Mutex
	Tasks                = make(map[int]service.Task)
	Calculations         = []service.Calculation{}
	BeingCalculated      = []service.Calculation{}
)

func TaskPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	staticPath := filepath.Join("..", "..", "static", "index.html")
	http.ServeFile(w, r, staticPath)
}

func AddTask(w http.ResponseWriter, r *http.Request) {
	NewTask := service.Task{}
	if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "application/json" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err = json.Unmarshal(body, &NewTask); err != nil {
			fmt.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if calculate.IsFloat(NewTask.Expression) {
			http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
			return
		}

		if err = calculate.ValidateInfixExpression(NewTask.Expression); err != nil {
			log.Printf("%v\n", err)
			http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
			return
		}

		tasksMutex.Lock()
		if _, ok := Tasks[NewTask.Id]; ok {
			tasksMutex.Unlock()
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		NewTask.Status = "In Process"
		NewTask.Original_Expression = NewTask.Expression
		Tasks[NewTask.Id] = NewTask
		tasksMutex.Unlock()

		newRPN, err := calculate.InfixToRPN(NewTask.Expression)
		log.Println(newRPN)

		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "{}")

		go func() {
			calculationsMutex.Lock()
			defer calculationsMutex.Unlock()
			calculate.RPNtoSeparateCalculations(newRPN, NewTask.Id, &Calculations, BeingCalculated)
		}()
	} else {
		http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
		return
	}
}

func HandleCalculations(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		calculationsMutex.Lock()
		defer calculationsMutex.Unlock()

		if len(Calculations) > 0 {
			calculation := Calculations[0]
			Calculations = Calculations[1:]

			beingCalculatedMutex.Lock()
			BeingCalculated = append(BeingCalculated, calculation)
			beingCalculatedMutex.Unlock()

			task_json, err := json.Marshal(calculation)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, string(task_json))
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	} else if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "application/json" {
		FinishedCalculation := service.Calculation{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err = json.Unmarshal(body, &FinishedCalculation); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if FinishedCalculation.Status != "Finished" && FinishedCalculation.Status != "Error" {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "{}")

		beingCalculatedMutex.Lock()
		service.DeleteCalculationFromSlice(FinishedCalculation, &BeingCalculated)
		beingCalculatedMutex.Unlock()

		tasksMutex.Lock()
		LinkedTask := Tasks[FinishedCalculation.Task_id]
		tasksMutex.Unlock()

		if LinkedTask.Status == "Finished" {
			return
		}

		if FinishedCalculation.Status == "Error" {
			log.Println("INSIDE")
			LinkedTask.Result = 0
			LinkedTask.Status = "Calculation Error"
			tasksMutex.Lock()
			Tasks[FinishedCalculation.Task_id] = LinkedTask
			tasksMutex.Unlock()
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
		tasksMutex.Lock()
		Tasks[FinishedCalculation.Task_id] = LinkedTask
		tasksMutex.Unlock()

		if calculate.IsFloat(LinkedTask.Expression) {
			res, _ := strconv.Atoi(LinkedTask.Expression)
			LinkedTask.Status = "Finished"
			LinkedTask.Result = res
			tasksMutex.Lock()
			Tasks[FinishedCalculation.Task_id] = LinkedTask
			tasksMutex.Unlock()
			log.Printf("FINISHED CALCULATING RESULT IS %d\n", LinkedTask.Result)
		} else {
			go func() {
				calculationsMutex.Lock()
				defer calculationsMutex.Unlock()
				calculate.RPNtoSeparateCalculations(LinkedExpressionRPN, LinkedTask.Id, &Calculations, BeingCalculated)
			}()
		}
	} else {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}
}

func HandleAllExpressions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		if r.Method == http.MethodGet {
			tasksMutex.Lock()
			all_expressions := make([]service.Task, 0, len(Tasks))
			for _, value := range Tasks {
				all_expressions = append(all_expressions, value)
			}
			tasksMutex.Unlock()

			expressions, err := json.Marshal(all_expressions)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, string(expressions))
		} else {
			http.Error(w, "Bad Request", http.StatusBadRequest)
		}
	} else if calculate.IsFloat(id) {
		searchedTaskId, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		tasksMutex.Lock()
		searchedTask, ok := Tasks[searchedTaskId]
		tasksMutex.Unlock()

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

func HandleRegistration(w http.ResponseWriter, r *http.Request) {
	// If it's not a POST request, we don't want it.
	if r.Method != http.MethodPost {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Opening a connection to the db and creating the tables if necessary.
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// This is important!
	// Foreign keys might not always be on by default.
	// However, we heavily rely on them, so if they don't work, we're toast.
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal("Failed to enable foreign key constraints:", err)
	}	

	createUserTableSQL := `
    CREATE TABLE IF NOT EXISTS users (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,       
        "name" TEXT UNIQUE,
        "password" TEXT
    );`

	_, err = db.Exec(createUserTableSQL)

	if err != nil {
		log.Fatal(err)
	}

	createExpressionTableSQL := `
	CREATE TABLE IF NOT EXISTS expressions (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"status" TEXT NOT NULL,
		"original_expression" TEXT NOT NULL,
		"expression" TEXT NOT NULL,
		"result" INTEGER,
		"owner" INTEGER,
		FOREIGN KEY(owner) REFERENCES users(id)
	);`

	_, err = db.Exec(createExpressionTableSQL)

	if err != nil {
		log.Fatal(err)
	}

	createTasksTableSQL := `
	CREATE TABLE IF NOT EXISTS tasks (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"RPN_string" TEXT,
		"status" TEXT,
		"Result" TEXT,
		"task_id" INTEGER,
		FOREIGN KEY(task_id) REFERENCES expressions(id)
	);`

	_, err = db.Exec(createTasksTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	newUser := service.User{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(body, &newUser); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Check if user already exists

	checkUserSQL := `SELECT name FROM users WHERE name = ?`
	var foundName string

	err = db.QueryRow(checkUserSQL, newUser.Name).Scan(&foundName)

	// This means that a user with that name was found
	if err == nil {
		http.Error(w, "User already exists. 409 Conflict.", http.StatusConflict)
		return
	}

	// If no errors encountered to this point, then we can try to add the user to the db
	addUserSQL := `INSERT INTO users (name, password) VALUES (?, ?)`
	_, err = db.Exec(addUserSQL, newUser.Name, newUser.Password)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	log.Printf("User %s created successfully!", newUser.Name)
	// ...and that's it! :)
}
