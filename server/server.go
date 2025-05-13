package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

// Constants
const (
	serverAddress = "localhost:8080"
	tickRate      = 100 * time.Millisecond //  Update game state every 100ms
	gameDuration  = 3 * time.Minute
)

// Data Structures (from previous version, repeated for completeness)
type Player struct {
	Username string
	Password string
	EXP      int
	Level    int
	Towers   []Tower
	Troops   []Troop
	Conn     net.Conn // Add this to associate player with connection
	Mana     int
}

type Tower struct {
	Type string
	HP   int
	ATK  int
	DEF  int
	CRIT int
	EXP  int
}

type Troop struct {
	Name    string
	HP      int
	ATK     int
	DEF     int
	MANA    int
	EXP     int
	Special string
}

type GameState struct {
	Player1   *Player
	Player2   *Player
	StartTime time.Time
	gameOver  bool
}

// Global variables
var (
	troopsData  []Troop
	towersData  []Tower
	activeGames = make(map[string]*GameState) // Map game ID to GameState.  Use string as key.
	players     = make(map[string]*Player)    // Map username to Player.
)

// Helper Functions (from previous version, repeated for completeness)
func loadJSONData(filename string, target interface{}) error {
	file, err := ioutil.ReadFile(filename) // Corrected: Use ioutil.ReadFile
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, target)
	if err != nil {
		return err
	}
	return nil
}

// calculateDamage calculates damage, including CRIT.
func calculateDamage(attacker interface{}, defender interface{}, critChance int) int {
	var attack, defense int
	switch a := attacker.(type) {
	case Tower:
		attack = a.ATK
	case Troop:
		attack = a.ATK
	case *Tower: // Handle pointers
		attack = a.ATK
	case *Troop:
		attack = a.ATK
	default:
		return 0 // Or handle the error as appropriate
	}

	switch d := defender.(type) {
	case Tower:
		defense = d.DEF
	case Troop:
		defense = d.DEF
	case *Tower: //handle pointers.
		defense = d.DEF
	case *Troop:
		defense = d.DEF
	default:
		return 0
	}

	damage := attack - defense
	if damage < 0 {
		return 0
	}
	//check for crit
	if critChance > 0 {
		randNum := rand.Intn(100)
		if randNum < critChance {
			damage = int(float64(attack)*1.2) - defense
			if damage < 0 {
				return 0
			}
		}
	}

	return damage
}

// updateGameState updates the game state, including damage calculation and mana regeneration.  This is called every tick.
func updateGameState(game *GameState) {
	if game.gameOver {
		return
	}
	// Calculate damage between troops and towers.
	//  This is a simplified version.  A real game would have much more complex logic
	//  for targeting, movement, and special abilities.
	player1 := game.Player1
	player2 := game.Player2

	//check if game time is over
	if time.Since(game.StartTime) >= gameDuration {
		game.gameOver = true
		determineWinner(game)
		return
	}

	//Regenerate Mana
	player1.Mana = min(player1.Mana+1, 10)
	player2.Mana = min(player2.Mana+1, 10)

	// Example:  Damage first tower of each player.  Simplified for demonstration.
	if len(player1.Towers) > 0 && len(player2.Towers) > 0 {
		damageToP1Tower := calculateDamage(&player2.Troops[0], &player1.Towers[0], player2.Towers[0].CRIT) //Simplified to first troop
		player1.Towers[0].HP -= damageToP1Tower
		damageToP2Tower := calculateDamage(&player1.Troops[0], &player2.Towers[0], player1.Towers[0].CRIT) //Simplified to first troop.
		player2.Towers[0].HP -= damageToP2Tower

		if player1.Towers[0].HP <= 0 {
			player1.Towers[0].HP = 0
			//award exp
			awardEXP(player2, player1.Towers[0].EXP)
		}
		if player2.Towers[0].HP <= 0 {
			player2.Towers[0].HP = 0
			awardEXP(player1, player2.Towers[0].EXP)
		}
	}

	// Check for game over (King Tower destroyed)
	if player1.Towers[0].HP <= 0 { //Index 0 is King Tower
		game.gameOver = true
		sendGameOverMessage(game, player2.Username)
		awardWinEXP(game, player2)
		return
	}
	if player2.Towers[0].HP <= 0 {
		game.gameOver = true
		sendGameOverMessage(game, player1.Username)
		awardWinEXP(game, player1)
		return
	}
	sendGameState(game) //send game state.
}

func awardWinEXP(game *GameState, winner *Player) {
	winner.EXP += 30
	levelUp(winner)
	savePlayerData(winner)
}

func awardDrawEXP(game *GameState) {
	game.Player1.EXP += 10
	levelUp(game.Player1)
	savePlayerData(game.Player1)
	game.Player2.EXP += 10
	levelUp(game.Player2)
	savePlayerData(game.Player2)
}

func awardEXP(player *Player, exp int) {
	player.EXP += exp
}

func levelUp(player *Player) {
	levelThreshold := 100 + player.Level*10 // Example: 100, 110, 120, ...
	if player.EXP >= levelThreshold {
		player.Level++
		// Increase troop/tower stats (simplified)
		for i := range player.Towers {
			player.Towers[i].HP = int(float64(player.Towers[i].HP) * 1.1)
			player.Towers[i].ATK = int(float64(player.Towers[i].ATK) * 1.1)
			player.Towers[i].DEF = int(float64(player.Towers[i].DEF) * 1.1)
		}
		for i := range player.Troops {
			player.Troops[i].HP = int(float64(player.Troops[i].HP) * 1.1)
			player.Troops[i].ATK = int(float64(player.Troops[i].ATK) * 1.1)
			player.Troops[i].DEF = int(float64(player.Troops[i].DEF) * 1.1)
		}
		fmt.Printf("Player %s leveled up to level %d!\n", player.Username, player.Level)
	}
}

func determineWinner(game *GameState) string {
	if !game.gameOver {
		return ""
	}
	p1TowersDestroyed := 3 - len(game.Player1.Towers)
	p2TowersDestroyed := 3 - len(game.Player2.Towers)

	if p1TowersDestroyed > p2TowersDestroyed {
		return game.Player1.Username
	} else if p2TowersDestroyed > p1TowersDestroyed {
		return game.Player2.Username
	} else {
		return "draw"
	}
}

// Simplified version.
func handleAuthentication(conn net.Conn, username, password string) (*Player, error) {
	// In a real application, you would validate the password against a stored hash.
	// For this example, we'll just check for a matching username and password.

	player, ok := players[username]
	if !ok {
		// Player doesn't exist, create a new player.
		player = &Player{
			Username: username,
			Password: password,
			EXP:      0,
			Level:    1,
			Towers:   towersData, // Initialize towers.  Make sure towersData is loaded.
			Troops:   troopsData, // Initialize troops. Make sure troopsData is loaded.
			Conn:     conn,
			Mana:     5,
		}
		players[username] = player //store the player
		savePlayerData(player)
		return player, nil
	}

	if player.Password != password {
		return nil, fmt.Errorf("incorrect password")
	}
	//update the connection.
	player.Conn = conn
	return player, nil
}

func initializeGame(player1, player2 *Player) *GameState {
	gameID := player1.Username + "-" + player2.Username // Simple game ID.
	game := &GameState{
		Player1:   player1,
		Player2:   player2,
		StartTime: time.Now(), //set the start time.
		gameOver:  false,
	}
	activeGames[gameID] = game
	return game
}

// saveData saves player data to a JSON file.
func savePlayerData(player *Player) error {
	filename := fmt.Sprintf("%s.json", player.Username)
	file, err := json.MarshalIndent(player, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, file, 0644) // Corrected: Use ioutil.WriteFile
}

// loadPlayerData loads player data from a JSON file.
func loadPlayerData(filename string) (Player, error) {
	file, err := ioutil.ReadFile(filename) // Corrected: Use ioutil.ReadFile
	if err != nil {
		if os.IsNotExist(err) {
			return Player{}, nil // Return default if file doesn't exist
		}
		return Player{}, err
	}
	var player Player
	err = json.Unmarshal(file, &player)
	if err != nil {
		return Player{}, err
	}
	return player, nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read username and password.
	reader := bufio.NewReader(conn)
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Error reading username:", err)
		return
	}
	username = strings.TrimSpace(username)

	password, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Error reading password:", err)
		return
	}
	password = strings.TrimSpace(password)

	// Authenticate the player.
	player, err := handleAuthentication(conn, username, password)
	if err != nil {
		fmt.Fprintln(conn, "Authentication failed")
		log.Println("Authentication failed:", err)
		return
	}
	fmt.Fprintln(conn, "Authentication successful")

	//For simplicity, auto match
	if len(activeGames) > 0 {
		for _, game := range activeGames {
			if game.Player2 == nil {
				game.Player2 = player
				game.StartTime = time.Now()
				fmt.Fprintln(game.Player1.Conn, "Game started")
				fmt.Fprintln(game.Player2.Conn, "Game started")
				return
			}
		}
	}

	// Start the game.
	// For simplicity, we'll start a game with another player who is waiting.
	// In a real application, you would implement a proper matchmaking system.
	var game *GameState
	if len(activeGames) == 0 {
		// No other games running, create a new one.
		game = initializeGame(player, nil) //player 1
		fmt.Fprintln(conn, "Waiting for another player...")
	} else {
		//connect to a game.
		for _, g := range activeGames {
			if g.Player2 == nil {
				g.Player2 = player
				game = g
				break
			}
		}
		if game == nil {
			game = initializeGame(player, nil) //player 1
			fmt.Fprintln(conn, "Waiting for another player...")
			return
		}

		fmt.Fprintln(game.Player1.Conn, "Game started with ", player.Username)
		fmt.Fprintln(game.Player2.Conn, "Game started with ", player.Username)
	}

	// Game loop.  This should run in a goroutine.
	go func() {
		ticker := time.NewTicker(tickRate)
		defer ticker.Stop()
		for !game.gameOver {
			<-ticker.C
			updateGameState(game)
		}
		// Game is over, clean up.
		gameID := game.Player1.Username + "-" + game.Player2.Username
		delete(activeGames, gameID)
		fmt.Println("Game", gameID, "is over")
		if game.Player1 != nil {
			savePlayerData(game.Player1)
		}
		if game.Player2 != nil {
			savePlayerData(game.Player2)
		}

	}()

	// Handle player commands (e.g., deploying troops).  This is a simplified example.
	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("Player", player.Username, "disconnected")
				return //normal disconnection
			}
			log.Println("Error reading from connection:", err)
			return
		}
		message = strings.TrimSpace(message)
		log.Println("Received from", player.Username, ":", message)

		// Example command:  "deploy troopName"
		parts := strings.Split(message, " ")
		if len(parts) > 1 && parts[0] == "deploy" {
			troopName := parts[1]
			// Find the troop in the available troops.
			var troopToDeploy *Troop
			for i := range player.Troops {
				if player.Troops[i].Name == troopName {
					troopToDeploy = &player.Troops[i]
					break
				}
			}
			if troopToDeploy != nil {
				if player.Mana >= troopToDeploy.MANA {
					// Deduct mana and deploy the troop (simplified).
					player.Mana -= troopToDeploy.MANA
					fmt.Println("Deploying troop", troopName, "for player", player.Username)
					// In a real game, you would add the troop to the game state
					// and handle its movement and attacks.
				} else {
					fmt.Fprintln(conn, "Not enough mana to deploy", troopName)
				}

			} else {
				fmt.Fprintln(conn, "Troop", troopName, "not found")
			}
		}
	}
}

func sendGameState(game *GameState) {
	// Create a simplified representation of the game state to send to clients.
	//  This avoids sending the entire GameState struct, which might contain sensitive
	//  or irrelevant information.  Send only the data needed by the client to display
	//  the current state of the game.

	p1Towers := make([]Tower, len(game.Player1.Towers))
	for i, tower := range game.Player1.Towers {
		p1Towers[i] = Tower{Type: tower.Type, HP: tower.HP}
	}
	p2Towers := make([]Tower, len(game.Player2.Towers))
	for i, tower := range game.Player2.Towers {
		p2Towers[i] = Tower{Type: tower.Type, HP: tower.HP}
	}

	gameStateData := map[string]interface{}{
		"player1": map[string]interface{}{
			"username": game.Player1.Username,
			"level":    game.Player1.Level,
			"towers":   p1Towers, // Include tower data
			"mana":     game.Player1.Mana,
		},
		"player2": map[string]interface{}{
			"username": game.Player2.Username,
			"level":    game.Player2.Level,
			"towers":   p2Towers, // Include tower data
			"mana":     game.Player2.Mana,
		},
		"gameOver":      game.gameOver,
		"timeRemaining": int(gameDuration - time.Since(game.StartTime)),
	}

	// Send the game state to both players.
	p1JSON, err := json.Marshal(gameStateData)
	if err != nil {
		log.Println("Error marshaling game state:", err)
		return
	}
	if game.Player1 != nil {
		fmt.Fprintln(game.Player1.Conn, string(p1JSON))
	}

	p2JSON, err := json.Marshal(gameStateData)
	if err != nil {
		log.Println("Error marshaling game state:", err)
		return
	}
	if game.Player2 != nil {
		fmt.Fprintln(game.Player2.Conn, string(p2JSON))
	}
}

func sendGameOverMessage(game *GameState, winner string) {
	message := fmt.Sprintf("Game Over! Winner: %s", winner)
	if game.Player1 != nil {
		fmt.Fprintln(game.Player1.Conn, message)
	}
	if game.Player2 != nil {
		fmt.Fprintln(game.Player2.Conn, message)
	}

}
func main() {
	// Load troop and tower data from JSON files.
	if err := loadJSONData("./troops.json", &troopsData); err != nil {
		log.Fatalf("Error loading troop data: %v", err)
	}
	if err := loadJSONData("./towers.json", &towersData); err != nil {
		log.Fatalf("Error loading tower data: %v", err)
	}

	// Load player data
	// Example of loading player data on server startup (you might do this on login instead)
	/*
	   player1Data, err := loadPlayerData("player1.json")
	   if err != nil {
	       log.Println("Error loading player data:", err) // Non-fatal error
	   }
	   if player1Data.Username != "" {
	       players[player1Data.Username] = &player1Data
	   }
	*/

	// Start the server.
	listener, err := net.Listen("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer listener.Close()
	fmt.Println("Server listening on", serverAddress)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue // Accept other connections.
		}
		// Handle each connection in a separate goroutine.
		go handleConnection(conn)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
