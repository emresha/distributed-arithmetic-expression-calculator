/*
This is the Agent.
His job is to constantly ask the orchestrator for tasks to calculate.
If there are none, the Agent patiently waits.
*/

package main

import (
	"bytes"
	calculate "distributed-calculator/internal/logic"
	"distributed-calculator/internal/service"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default values for variables
var COMPUTING_POWER int = 10
var TIME_ADDITION_MS int = 5000
var TIME_SUBTRACTION_MS int = 5000
var TIME_MULTIPLICATIONS_MS int = 15000
var TIME_DIVISIONS_MS int = 15000
var Calculations = make(chan service.Calculation)

func Calculate() {
	for {
		select {
		case calc := <-Calculations:
			var sleepDuration time.Duration

			switch {
			case strings.Contains(calc.RPN_string, "+"):
				sleepDuration = time.Duration(TIME_ADDITION_MS) * time.Millisecond
			case strings.Contains(calc.RPN_string, "-"):
				sleepDuration = time.Duration(TIME_SUBTRACTION_MS) * time.Millisecond
			case strings.Contains(calc.RPN_string, "*"):
				sleepDuration = time.Duration(TIME_MULTIPLICATIONS_MS) * time.Millisecond
			case strings.Contains(calc.RPN_string, "/"):
				sleepDuration = time.Duration(TIME_DIVISIONS_MS) * time.Millisecond
			default:
				log.Println("Unsupported operation encountered.")
				continue
			}

			time.Sleep(sleepDuration)
			log.Println(calc.RPN_string)
			tokens := strings.Split(calc.RPN_string, " ")
			result, err := calculate.EvalRPN(tokens)
			if err != nil {
				log.Println("Division by zero encountered.")
				calc.Status = "Error"
				calc.Result = result
			} else {
				calc.Status = "Finished"
				calc.Result = result
				log.Printf("Finished calculation %d. Took %d ms.\n", calc.Task_id, sleepDuration.Milliseconds())
			}

			calc_json, err := json.Marshal(calc)
			if err != nil {
				log.Println("Error marshalling json.")
				continue
			}

			req, err := http.NewRequest("POST", "http://localhost:8080/internal/task", bytes.NewBuffer(calc_json))
			if err != nil {
				log.Println("Error creating request.")
				continue
			}

			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Println("Error sending request.")
				continue
			}

			if resp.StatusCode != http.StatusOK {
				log.Printf("POST Request unsuccessful. Status Code: %d\n", resp.StatusCode)
			} else {
				log.Println("Successfully sent the POST Request.")
			}
			resp.Body.Close()
		}
	}
}
func main() {

	// Getting environment variables.
	COMPUTING_POWER_STR := os.Getenv("COMPUTING_POWER")
	TIME_ADDITION_MS_STR := os.Getenv("TIME_ADDITION_MS")
	TIME_SUBTRACTION_MS_STR := os.Getenv("TIME_SUBTRACTION_MS")
	TIME_MULTIPLICATIONS_MS_STR := os.Getenv("TIME_MULTIPLICATIONS_MS")
	TIME_DIVISIONS_MS_STR := os.Getenv("TIME_DIVISIONS_MS")

	log.Println("The Agent is being launched...")

	if COMPUTING_POWER_STR == "" {
		// Setting a default value
		log.Println("COMPUTING_POWER NOT SET. DEFAULT VALUE WAS SET.")
		fmt.Printf("Computing power is: %d\n", COMPUTING_POWER)
	} else {
		COMPUTING_POWER, err := strconv.Atoi(COMPUTING_POWER_STR)
		if err != nil {
			log.Println("ERROR FETCHING ENV VARIABLE COMPUTING POWER. ASSUMING 10.")
		}
		fmt.Printf("Computing power is: %d\n", COMPUTING_POWER)
	}

	if TIME_ADDITION_MS_STR == "" {
		// Setting a default value
		log.Println("TIME_ADDITION_MS NOT SET. DEFAULT VALUE WAS SET.")
		fmt.Printf("Addition time is: %d ms.\n", TIME_ADDITION_MS)
	} else {
		TIME_ADDITION_MS, err := strconv.Atoi(TIME_ADDITION_MS_STR)
		if err != nil {
			log.Println("ERROR FETCHING ENV VARIABLE TIME_ADDITION_MS. ASSUMING 5000.")
		}
		fmt.Printf("Computing power is: %d\n", TIME_ADDITION_MS)
	}

	if TIME_SUBTRACTION_MS_STR == "" {
		// Setting a default value
		log.Println("TIME_SUBTRACTION_MS NOT SET. DEFAULT VALUE WAS SET.")
		fmt.Printf("Subtraction time is: %d ms.\n", TIME_SUBTRACTION_MS)
	} else {
		TIME_SUBTRACTION_MS, err := strconv.Atoi(TIME_SUBTRACTION_MS_STR)
		if err != nil {
			log.Println("ERROR FETCHING ENV VARIABLE TIME_SUBTRACTION_MS. ASSUMING 5000.")
		}
		fmt.Printf("Computing power is: %d\n", TIME_SUBTRACTION_MS)
	}

	if TIME_MULTIPLICATIONS_MS_STR == "" {
		// Setting a default value
		log.Println("TIME_MULTIPLICATIONS_MS NOT SET. DEFAULT VALUE WAS SET.")
		fmt.Printf("Multiplication time is: %d ms.\n", TIME_MULTIPLICATIONS_MS)
	} else {
		TIME_MULTIPLICATIONS_MS, err := strconv.Atoi(TIME_MULTIPLICATIONS_MS_STR)
		if err != nil {
			log.Println("ERROR FETCHING ENV VARIABLE TIME_MULTIPLICATIONS_MS. ASSUMING 15000.")
		}
		fmt.Printf("Computing power is: %d\n", TIME_MULTIPLICATIONS_MS)
	}

	if TIME_DIVISIONS_MS_STR == "" {
		// Setting a default value
		log.Println("TIME_DIVISIONS_MS NOT SET. DEFAULT VALUE WAS SET.")
		fmt.Printf("Division time is: %d ms.\n", TIME_DIVISIONS_MS)
	} else {
		TIME_DIVISIONS_MS, err := strconv.Atoi(TIME_DIVISIONS_MS_STR)
		if err != nil {
			log.Println("ERROR FETCHING ENV VARIABLE TIME_DIVISIONS_MS. ASSUMING 15000.")
		}
		fmt.Printf("Computing power is: %d\n", TIME_DIVISIONS_MS)
	}

	for i := 0; i < COMPUTING_POWER; i++ {
		go Calculate()
	}

	client := &http.Client{}

	// Infinite loop that sends requests to the Orchestrator.
	for {
		time.Sleep(5 * time.Second)
		req, err := http.NewRequest("GET", "http://localhost:8080/internal/task", nil)
		if err != nil {
			log.Printf("GET Request error: %v\n", err)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("GET Request error: %v\n", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("GET Request unsuccessful, status code: %d\n", resp.StatusCode)
		} else {
			log.Println("GET Request successful.")
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Println("Response Body read error.")
				continue
			}

			calc := service.Calculation{}
			json.Unmarshal(body, &calc)
			Calculations <- calc

			resp.Body.Close()
		}


	}
}
