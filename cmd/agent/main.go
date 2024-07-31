/*
This is the Agent.
His job is to constantly ask the orchestrator for tasks to calculate.
If there are none, the Agent patiently waits.
*/

package main

import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"distributed-calculator/config"
	calculate "distributed-calculator/internal/logic"
	"distributed-calculator/internal/service"
	pb "distributed-calculator/proto"
	"log"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Default values for variables
var COMPUTING_POWER int = 10
var TIME_ADDITION_MS int = 5000
var TIME_SUBTRACTION_MS int = 5000
var TIME_MULTIPLICATIONS_MS int = 15000
var TIME_DIVISIONS_MS int = 15000
var Calculations = make(chan service.Calculation)
var grpcClient pb.CalculatorServiceClient

func Calculate() {
	for {
		select {
		case calc := <-Calculations:
			var sleepDuration time.Duration
			RPNSlice := strings.Split(calc.RPN_string, " ")
			switch {
			case slices.Contains(RPNSlice, "+"):
				sleepDuration = time.Duration(TIME_ADDITION_MS) * time.Millisecond
			case slices.Contains(RPNSlice, "-"):
				sleepDuration = time.Duration(TIME_SUBTRACTION_MS) * time.Millisecond
			case slices.Contains(RPNSlice, "*"):
				sleepDuration = time.Duration(TIME_MULTIPLICATIONS_MS) * time.Millisecond
			case slices.Contains(RPNSlice, "/"):
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

			_, err = grpcClient.SendCalculation(context.TODO(), &pb.SendCalculationRequest{
				TaskId:    int64(calc.Task_id),
				RPNString: calc.RPN_string,
				Status:    calc.Status,
				Result:    int64(calc.Result),
			})

			if err != nil {
				log.Printf("gRPC send error: %v", err)
			}
		}
	}
}
func main() {
	log.Println("The Agent is being launched...")
	// Getting environment variables.
	cfg, err := config.LoadConfig(filepath.Join("..", "..", "config.cfg"))
	if err != nil {
		log.Printf("ERROR READING CONFIG FILE: %v. DEFAULT VALUES WERE SET.\n", err)
	} else {
		COMPUTING_POWER = cfg.ComputingPower
		TIME_ADDITION_MS = cfg.TimeAdditionMs
		TIME_SUBTRACTION_MS = cfg.TimeSubtractionMs
		TIME_MULTIPLICATIONS_MS = cfg.TimeMultiplicationsMs
		TIME_DIVISIONS_MS = cfg.TimeDivisionsMs
	}

	log.Printf("Computing power is: %d\n", COMPUTING_POWER)
	log.Printf("Addition time is: %d ms.\n", TIME_ADDITION_MS)
	log.Printf("Subtraction time is: %d ms.\n", TIME_SUBTRACTION_MS)
	log.Printf("Multiplication time is: %d ms.\n", TIME_MULTIPLICATIONS_MS)
	log.Printf("Division time is: %d ms.\n", TIME_DIVISIONS_MS)

	// Opening a connection to the db and creating the tables if necessary.
	db, err := sql.Open("sqlite3", "../orchestrator/data.db")
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

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Couldn't connect to gRPC: %v", err)
	}

	log.Println("Successfully connected to gRPC on :50051...")
	defer conn.Close()

	grpcClient = pb.NewCalculatorServiceClient(conn)

	for i := 0; i < COMPUTING_POWER; i++ {
		go Calculate()
	}

	// Infinite loop that sends requests to the Orchestrator.
	for {
		time.Sleep(1 * time.Second)

		gRPC_Calculation, err := grpcClient.GetCalculation(context.TODO(), &pb.GetCalculationRequest{})
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("gRPC client error: no tasks.")
			} else {
				log.Printf("gRPC client error: %v", err)
			}
		} else {
			newCalc := service.Calculation{
				Task_id:    int(gRPC_Calculation.TaskId),
				RPN_string: gRPC_Calculation.RPNString,
				Status:     gRPC_Calculation.Status,
				Result:     int(gRPC_Calculation.Result),
			}
			_, err = db.Exec("UPDATE tasks SET status = 'In Process' WHERE task_id = ? AND RPN_string = ?", newCalc.Task_id, newCalc.RPN_string)
			if err != nil {
				log.Printf("Calculation update error, skipping. %v", err)
				continue
			}
			Calculations <- newCalc
		}

	}
}
