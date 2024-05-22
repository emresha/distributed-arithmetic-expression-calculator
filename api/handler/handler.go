package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"distributed-calculator/logic/expressions"
)

type Task struct {
	Id         int    `json:"id"`
	Expression string `json:"expression"`
}

func AddCalculation(w http.ResponseWriter, r *http.Request) {
	new := Task{}
	if r.Method == http.MethodPost && r.Header["Content-Type"][0] == "application/json" {
		body, err := io.ReadAll(r.Body)
		// handle error
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		

		if err = json.Unmarshal(body, &new); err != nil {
			fmt.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		
		if err = expressions.ValidateInfixExpression(new.Expression); err != nil {
			http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "{}")
	} else {
		http.Error(w, "Bad Request", http.StatusUnprocessableEntity)
		return
	}
}
