package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

const serverAddress = "localhost:8080" // Or as a command-line argument

func main() {
	// Connect to the server.
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Read username and password from the user.
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Send username and password to the server.
	fmt.Fprintln(conn, username)
	fmt.Fprintln(conn, password)

	// Read the server's response.
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read response from server: %v", err)
	}
	fmt.Println(response)

	if response == "Authentication failed\n" {
		return
	}

	// Game loop.
	for {
		// Read game state from the server.
		gameStateJSON, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("Server disconnected.")
				return
			}
			log.Fatalf("Error reading game state: %v", err)
		}

		// Unmarshal the JSON data.
		var gameState map[string]interface{}
		err = json.Unmarshal([]byte(gameStateJSON), &gameState)
		if err != nil {
			log.Printf("Error unmarshalling game state: %v, data: %s\n", err, gameStateJSON)
			continue // Skip this iteration, try to read again.  Don't crash.
		}

		// Display the game state.
		displayGameState(gameState)

		// Get player input (e.g., deploy a troop).
		fmt.Print("Enter command (e.g., deploy Pawn): ")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)

		// Send the command to the server.
		fmt.Fprintln(conn, command)
	}
}

func displayGameState(gameState map[string]interface{}) {
	// Clear the console (cross-platform way is complex, simplified here).
	//  For Windows, you might use "cls", for others "clear".
	fmt.Print("\033[H\033[2J") // Clear screen for Unix-like systems

	fmt.Println("--- Game State ---")

	// Display Player 1 information.
	player1, ok := gameState["player1"].(map[string]interface{})
	if ok {
		fmt.Println("Player 1:")
		fmt.Println("  Username:", player1["username"])
		fmt.Println("  Level:", player1["level"])
		fmt.Println("  Mana:", player1["mana"])
		towers, ok := player1["towers"].([]interface{})
		if ok {
			fmt.Println("  Towers:")
			for _, tower := range towers {
				towerMap, ok := tower.(map[string]interface{})
				if ok {
					fmt.Printf("    Type: %s, HP: %v\n", towerMap["Type"], towerMap["HP"])
				}

			}
		}
	}

	// Display Player 2 information.
	player2, ok := gameState["player2"].(map[string]interface{})
	if ok {
		fmt.Println("Player 2:")
		fmt.Println("  Username:", player2["username"])
		fmt.Println("  Level:", player2["level"])
		fmt.Println("  Mana:", player2["mana"])
		towers, ok := player2["towers"].([]interface{})
		if ok {
			fmt.Println("  Towers:")
			for _, tower := range towers {
				towerMap, ok := tower.(map[string]interface{})
				if ok {
					fmt.Printf("    Type: %s, HP: %v\n", towerMap["Type"], towerMap["HP"])
				}

			}
		}
	}
	gameOver, ok := gameState["gameOver"].(bool)
	if ok {
		fmt.Println("Game Over: ", gameOver)
	}

	timeRemaining, ok := gameState["timeRemaining"].(float64)
	if ok {
		fmt.Printf("Time Remaining: %v seconds\n", timeRemaining)
	}
}
