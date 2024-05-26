/*
This is the Agent.
His job is to constantly ask the orchestrator for tasks to calculate.
If there are none, the Agent patiently waits.
*/

package main

import (
	"distributed-calculator/internal/service"
	"fmt"
	"log"
	"os"
	"strconv"
)

func Calculate(service.Calculation) {
	// ...
}

// Testing for now
func main() {
	var COMPUTING_POWER int
	COMPUTING_POWER_STR := os.Getenv("COMPUTING_POWER")
	if COMPUTING_POWER_STR == "" {
		// Setting a default value
		log.Println("COMPUTING_POWER NOT SET. DEFAULT VALUE WAS SET.")
		COMPUTING_POWER = 10
		fmt.Printf("Computing power is: %d\n", COMPUTING_POWER)
	} else {
		COMPUTING_POWER, err := strconv.Atoi(COMPUTING_POWER_STR)
		if err != nil {
			log.Println("ERROR FETCHING ENV VARIABLE COMPUTING POWER. ASSUMING 10.")
			COMPUTING_POWER = 10
		}
		fmt.Printf("Computing power is: %d\n", COMPUTING_POWER)
	}
}