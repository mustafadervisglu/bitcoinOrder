package main

import (
	"fmt"
	"github.com/labstack/gommon/log"
	"time"
)

func main() {

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		//fmt.Println("Checking for matches...")
		matches, err := orderRepository.CheckOrder()
		if err != nil {
			log.Errorf("Failed to check orders: %v", err)
			continue
		}
		if len(matches) > 0 {
			fmt.Printf("Matches found: %v\n", matches)
		} else {
			//fmt.Println("No matches found.")
		}
	}
}
