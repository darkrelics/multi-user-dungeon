package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/robinje/multi-user-dungeon/core"
)

func NewServer(config core.Configuration) (*core.Server, error) {
	// Initialize the server with the configuration
	server := &core.Server{
		Port:        config.Port,
		PlayerIndex: &core.Index{},
		Config:      config,
		StartTime:   time.Now(),
		Rooms:       make(map[int64]*core.Room),
		Characters:  make(map[string]*core.Character),
		Balance:     config.Balance,
		AutoSave:    config.AutoSave,
		Health:      config.Health,
		Essence:     config.Essence,
	}

	log.Printf("Initializing database...")

	// Initialize the database
	var err error
	server.Database, err = core.NewKeyPair(config.DataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// Establish the player index
	server.PlayerIndex.IndexID = 1

	// Load the character names from the database

	log.Printf("Loading character names from database...")

	server.CharacterExists, err = core.LoadCharacterNames(server.Database.db)
	if err != nil {
		log.Printf("Error loading character names from database: %v", err)
	}

	server.Archetypes, err = LoadArchetypes(server.Database.db)
	if err != nil {
		log.Printf("Error loading archetypes from database: %v", err)
	}

	// Add a default room

	server.Rooms[0] = NewRoom(0, "The Void", "The Void", "You are in a void of nothingness. If you are here, something has gone terribly wrong.")

	// Load rooms into the server

	log.Printf("Loading rooms from database...")

	server.Rooms, err = server.Database.LoadRooms()
	if err != nil {
		return nil, fmt.Errorf("failed to load rooms: %v", err)
	}

	return server, nil
}

func main() {
	// Read configuration file
	configFile := flag.String("config", "config.json", "Configuration file")
	flag.Parse()

	config, err := loadConfiguration(*configFile)
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
		return
	}

	server, err := NewServer(config)
	if err != nil {
		log.Printf("Failed to create server: %v", err)
		return
	}

	// Start the auto-save routine
	go AutoSave(server)

	// Start the server
	if err := server.StartSSHServer(); err != nil {
		log.Printf("Failed to start server: %v", err)
		return
	}
}

func loadConfiguration(configFile string) (Configuration, error) {
	var config Configuration

	data, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

type Index struct {
	IndexID uint64
	mu      sync.Mutex
}

func (i *Index) GetID() uint64 {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.IndexID++
	return i.IndexID
}

func (i *Index) SetID(id uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if id > i.IndexID {
		i.IndexID = id
	}
}

func AutoSave(server *core.Server) {
	for {
		// Sleep for the configured duration
		time.Sleep(time.Duration(server.AutoSave) * time.Minute)

		log.Println("Starting auto-save process...")

		// Save active characters
		if err := SaveActiveCharacters(); err != nil {
			log.Printf("Failed to save characters: %v", err)
		} else {
			log.Println("Active characters saved successfully")
		}

		// Save active rooms
		if err := server.SaveActiveRooms(); err != nil {
			log.Printf("Failed to save rooms: %v", err)
		} else {
			log.Println("Active rooms saved successfully")
		}

		// Save active items
		if err := server.SaveActiveItems(); err != nil {
			log.Printf("Failed to save items: %v", err)
		} else {
			log.Println("Active items saved successfully")
		}

		log.Println("Auto-save process completed")
	}
}
