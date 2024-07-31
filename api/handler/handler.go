package handler

import (
	"database/sql"
	calculate "distributed-calculator/internal/logic"
	"distributed-calculator/internal/service"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db                   *sql.DB
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

func AuthPage(w http.ResponseWriter, r *http.Request) {
	staticPath := filepath.Join("..", "..", "static", "auth.html")
	http.ServeFile(w, r, staticPath)
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	name, err := service.CheckAuthentication(r)
	if err != nil {
		log.Println(err) // named cookie not present
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := map[string]string{"username": name}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func AddTask(w http.ResponseWriter, r *http.Request) {
	name, err := service.CheckAuthentication(r)

	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	NewTask := service.Task{
		Owner: name,
	}

	// Opening a connection to the db and creating the tables if necessary.
	db, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}

	// This is important!
	// Foreign keys might not always be on by default.
	// However, we heavily rely on them, so if they don't work, we're toast.
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal("Failed to enable foreign key constraints:", err)
	}

	if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "application/json" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err = json.Unmarshal(body, &NewTask); err != nil {
			log.Println(err)
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

		var checkTask int

		err = db.QueryRow("SELECT id FROM expressions WHERE id = ?", NewTask.Id).Scan(checkTask)

		if err == nil {
			http.Error(w, "Conflict", http.StatusConflict)
			return
		}

		NewTask.Status = "In Process"
		NewTask.Original_Expression = NewTask.Expression

		var ownerID int
		err = db.QueryRow(`SELECT id FROM users WHERE name = ?`, name).Scan(&ownerID)

		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec(`INSERT INTO expressions (id, status, original_expression, expression, result, owner) VALUES (?, ?, ?, ?, ?, ?)`, NewTask.Id, NewTask.Status, NewTask.Original_Expression, NewTask.Expression, NewTask.Result, ownerID)

		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

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
			calculate.RPNtoSeparateCalculations(newRPN, NewTask.Id, db)
		}()
	} else {
		http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
		return
	}
}

func HandleAllExpressions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	name, err := service.CheckAuthentication(r)

	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Opening a connection to the db and creating the tables if necessary.
	db, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}

	// This is important!
	// Foreign keys might not always be on by default.
	// However, we heavily rely on them, so if they don't work, we're toast.
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal("Failed to enable foreign key constraints:", err)
	}

	var userId int
	err = db.QueryRow(`SELECT id FROM users WHERE name = ?`, name).Scan(&userId)

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if id == "" {
		if r.Method == http.MethodGet {
			rows, err := db.Query(`SELECT id, status, original_expression, expression, result, owner FROM expressions WHERE owner = ?`, userId)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			tasksMutex.Lock()
			all_expressions := make([]service.Task, 0, len(Tasks))
			for rows.Next() {
				var exp_id, owner, result int
				var status, original_expression, expression string
				err := rows.Scan(&exp_id, &status, &original_expression, &expression, &result, &owner)
				if err != nil {
					// We really shouldn't terminate the whole server if there is a faulty expression...
					log.Printf("A very bad error while retrieving all expressions: %v", err)
				}
				all_expressions = append(all_expressions, service.Task{
					Id:                  exp_id,
					Status:              status,
					Original_Expression: original_expression,
					Expression:          expression,
					Result:              result,
					Owner:               name,
				})
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

		var exp_id, owner, result int
		var status, original_expression, expression string

		err = db.QueryRow(`SELECT id, status, original_expression, expression, result, owner FROM expressions WHERE id = ?`, searchedTaskId).Scan(&exp_id, &status, &original_expression, &expression, &result, &owner)

		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		var userId int

		err = db.QueryRow(`SELECT id FROM users WHERE name = ?`, name).Scan(&userId)

		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if owner != userId {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		searchedTaskJson, err := json.Marshal(service.Task{
			Id:                  exp_id,
			Status:              status,
			Original_Expression: original_expression,
			Expression:          expression,
			Result:              result,
			Owner:               name,
		})

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

	// New user instance for unmarshalling.
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

	// Check if user already exists.
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

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := service.CheckAuthentication(r)
	if err == nil {
		http.Error(w, "Already signed in.", http.StatusBadRequest)
		return
	}

	// Opening a connection to the db and creating the tables if necessary.
	db, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Enable foreign key constraints.
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal("Failed to enable foreign key constraints:", err)
	}

	user := service.User{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(body, &user); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var storedPassword string
	err = db.QueryRow(`SELECT password FROM users WHERE name = ?`, user.Name).Scan(&storedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Check if passwords are the same.
	// I didn't hash them, sorry.
	if user.Password != storedPassword {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Create JWT token.
	expirationTime := time.Now().Add(15 * time.Minute)
	claims := jwt.MapClaims{
		"name": user.Name,
		"nbf":  time.Now().Unix(),
		"exp":  expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(service.JWTSecretToken)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return the token to the client.
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  expirationTime,
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
	})
}

func GiveTask() (service.Calculation, error) {
	// Opening a connection to the db and creating the tables if necessary.
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}

	// This is important!
	// Foreign keys might not always be on by default.
	// However, we heavily rely on them, so if they don't work, we're toast.
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal("Failed to enable foreign key constraints:", err)
	}

		// Retrieve a calculation from the database
		calculationsMutex.Lock()
		defer calculationsMutex.Unlock()

		var calculation service.Calculation
		err = db.QueryRow("SELECT task_id, RPN_string, status, result FROM tasks WHERE status = 'Waiting' LIMIT 1").Scan(
			&calculation.Task_id,
			&calculation.RPN_string,
			&calculation.Status,
			&calculation.Result,
		)

		if err != nil {
			return service.Calculation{}, err
		}

		return calculation, nil
}

func TakeTask(finishedCalculation service.Calculation) error {
		if finishedCalculation.Status != "Finished" && finishedCalculation.Status != "Error" {
			return errors.New("bad request")
		}

		// Update the calculation status in the database
		beingCalculatedMutex.Lock()
		_, err := db.Exec("UPDATE tasks SET status = ?, result = ? WHERE task_id = ? AND RPN_string = ?",
			finishedCalculation.Status, finishedCalculation.Result, finishedCalculation.Task_id, finishedCalculation.RPN_string)
		beingCalculatedMutex.Unlock()
		if err != nil {
			log.Printf("Failed to update calculation status: %v\n", err)
			return err
		}

		// Retrieve the linked task
		var linkedTask service.Task
		tasksMutex.Lock()
		err = db.QueryRow("SELECT id, status, original_expression, expression, result, owner FROM expressions WHERE id = ?", finishedCalculation.Task_id).Scan(
			&linkedTask.Id,
			&linkedTask.Status,
			&linkedTask.Original_Expression,
			&linkedTask.Expression,
			&linkedTask.Result,
			&linkedTask.Owner,
		)
		tasksMutex.Unlock()
		if err != nil {
			log.Printf("Failed to retrieve linked task: %v\n", err)
			return err
		}

		// Check if the task has already finished
		if linkedTask.Status == "Finished" {
			log.Printf("Task already finished error")
			return err
		}

		if finishedCalculation.Status == "Error" {
			linkedTask.Result = 0
			linkedTask.Status = "Calculation Error"
			tasksMutex.Lock()
			_, err := db.Exec("UPDATE expressions SET status = ?, result = ? WHERE id = ?", linkedTask.Status, linkedTask.Result, linkedTask.Id)
			tasksMutex.Unlock()
			if err != nil {
				log.Printf("Failed to update task status to error: %v\n", err)
			}
			return err
		}

		log.Printf("Finished calculation: %s, Result: %d\n", finishedCalculation.RPN_string, finishedCalculation.Result)
		log.Printf("Linked Task Expression infix: %s\n", linkedTask.Expression)
		linkedExpressionRPN, _ := calculate.InfixToRPN(linkedTask.Expression)
		log.Printf("Linked Task Expression RPN: %s\n", linkedExpressionRPN)
		linkedExpressionRPN = strings.ReplaceAll(linkedExpressionRPN, finishedCalculation.RPN_string, fmt.Sprintf("%d", finishedCalculation.Result))
		log.Printf("Linked Task Expression New RPN: %s\n", linkedExpressionRPN)
		linkedExpressionInfix, _ := calculate.RPNtoInfix(linkedExpressionRPN)
		log.Printf("Linked Task Expression New Infix: %s\n", linkedExpressionInfix)
		linkedTask.Expression = linkedExpressionInfix

		tasksMutex.Lock()
		_, err = db.Exec("UPDATE expressions SET expression = ? WHERE id = ?", linkedTask.Expression, linkedTask.Id)
		tasksMutex.Unlock()
		if err != nil {
			log.Printf("Failed to update linked task expression: %v\n", err)
			return err
		}

		if calculate.IsFloat(linkedTask.Expression) {
			res, _ := strconv.ParseFloat(linkedTask.Expression, 64)
			linkedTask.Status = "Finished"
			linkedTask.Result = int(res)
			tasksMutex.Lock()
			_, err := db.Exec("UPDATE expressions SET status = ?, result = ? WHERE id = ?", linkedTask.Status, linkedTask.Result, linkedTask.Id)
			tasksMutex.Unlock()
			if err != nil {
				log.Printf("Failed to mark task as finished: %v\n", err)
				return err
			}
			log.Printf("FINISHED CALCULATING RESULT IS %d\n", linkedTask.Result)
		} else {
			go func() {
				calculationsMutex.Lock()
				defer calculationsMutex.Unlock()
				calculate.RPNtoSeparateCalculations(linkedExpressionRPN, linkedTask.Id, db)
			}()
		}

		return nil
}