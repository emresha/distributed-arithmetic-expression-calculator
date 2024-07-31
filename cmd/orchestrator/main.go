package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"google.golang.org/grpc"
	"distributed-calculator/api/handler"
	"distributed-calculator/internal/service"
	pb "distributed-calculator/proto"
)

type Server struct {
	pb.CalculatorServiceServer
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) GetCalculation(ctx context.Context, in *pb.GetCalculationRequest) (*pb.GetCalculationResponse, error) {
	calc, err := handler.GiveTask()
	if err != nil {
		return &pb.GetCalculationResponse{}, err
	}

	return &pb.GetCalculationResponse{
		TaskId:    int64(calc.Task_id),
		RPNString: calc.RPN_string,
		Status:    calc.Status,
		Result:    int64(calc.Result),
	}, nil
}

func (s *Server) SendCalculation(ctx context.Context, out *pb.SendCalculationRequest) (*pb.SendCalculationResponse, error) {
	calc := service.Calculation{
		Task_id:    int(out.TaskId),
		RPN_string: out.RPNString,
		Status:     out.Status,
		Result:     int(out.Result),
	}

	err := handler.TakeTask(calc)
	if err != nil {
		return &pb.SendCalculationResponse{}, err
	}

	return &pb.SendCalculationResponse{}, nil
}

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
	mux.HandleFunc("/api/v1/register", handler.HandleRegistration)
	mux.HandleFunc("/api/v1/login", handler.HandleLogin)
	mux.HandleFunc("/auth", handler.AuthPage)
	mux.HandleFunc("/user", handler.UserHandler)

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

	go func() {
		log.Println("HTTP server running on port 8080...")
		http.ListenAndServe(":8080", mux)
	}()

	grpcServer := grpc.NewServer()
	pb.RegisterCalculatorServiceServer(grpcServer, NewServer())

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v\n", err)
	}
	log.Println("gRPC server running on port 50051...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v\n", err)
	}
}
