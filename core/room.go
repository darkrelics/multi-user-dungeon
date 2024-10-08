package core

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
)

// StoreRooms stores all rooms into the DynamoDB database.
func (kp *KeyPair) StoreRooms(rooms map[int64]*Room) error {
	for _, room := range rooms {
		err := kp.WriteRoom(room)
		if err != nil {
			Logger.Error("Error storing room", "room_id", room.RoomID, "error", err)
			return fmt.Errorf("error storing room %d: %w", room.RoomID, err)
		}
	}
	Logger.Info("Successfully stored all rooms")
	return nil
}

func (kp *KeyPair) LoadRooms() (map[int64]*Room, error) {
	rooms := make(map[int64]*Room)

	// Load room data
	var roomsData []RoomData
	err := kp.Scan("rooms", &roomsData)
	if err != nil {
		Logger.Error("Error scanning rooms", "error", err)
		return nil, fmt.Errorf("error scanning rooms: %w", err)
	}

	// First pass: create all rooms without exits or items
	for _, roomData := range roomsData {
		room := NewRoom(roomData.RoomID, roomData.Area, roomData.Title, roomData.Description)
		rooms[room.RoomID] = room
	}

	// Load all exits
	allExits, err := kp.LoadAllExits()
	if err != nil {
		Logger.Error("Error loading exits", "error", err)
		return nil, fmt.Errorf("error loading exits: %w", err)
	}

	// Load all items
	allItems, err := kp.LoadAllItems()
	if err != nil {
		Logger.Error("Error loading items", "error", err)
		return nil, fmt.Errorf("error loading items: %w", err)
	}

	// Second pass: add exits and items to rooms, and resolve target rooms
	for _, roomData := range roomsData {
		room, exists := rooms[roomData.RoomID]
		if !exists {
			Logger.Warn("Room not found in second pass", "room_id", roomData.RoomID)
			continue
		}

		// Add exits to the room
		if roomExits, exists := allExits[roomData.RoomID]; exists {
			room.Exits = roomExits
		}

		// Resolve TargetRoom pointers
		for _, exit := range room.Exits {
			if exit.TargetRoom == nil {
				Logger.Warn("Nil TargetRoom found", "room_id", roomData.RoomID, "direction", exit.Direction)
				continue
			}
			targetRoomID := exit.TargetRoom.RoomID
			if targetRoom, exists := rooms[targetRoomID]; exists {
				exit.TargetRoom = targetRoom
			} else {
				Logger.Warn("Target room not found for exit", "room_id", roomData.RoomID, "direction", exit.Direction, "target_room_id", targetRoomID)
			}
		}

		// Add items to the room
		room.Items = make(map[uuid.UUID]*Item)
		for _, itemID := range roomData.ItemIDs {
			if itemID == "" {
				Logger.Warn("Empty item ID found for room", "room_id", roomData.RoomID)
				continue
			}
			if item, exists := allItems[itemID]; exists {
				itemUUID, err := uuid.Parse(itemID)
				if err != nil {
					Logger.Error("Invalid item UUID", "item_id", itemID, "error", err)
					continue
				}
				room.Items[itemUUID] = item
			} else {
				Logger.Warn("Item not found for room", "room_id", roomData.RoomID, "item_id", itemID)
			}
		}
	}

	Logger.Info("Successfully loaded rooms from database", "count", len(rooms))
	return rooms, nil
}

// LoadExits retrieves all exits for a given room from the DynamoDB database.
func (kp *KeyPair) LoadExits(roomID int64) (map[string]*Exit, error) {
	exits := make(map[string]*Exit)

	var exitsData []ExitData

	// Prepare the query input
	keyCondition := "RoomID = :roomID"
	expressionAttributeValues := map[string]*dynamodb.AttributeValue{
		":roomID": {N: aws.String(strconv.FormatInt(roomID, 10))},
	}

	// Query the "exits" table for exits associated with the roomID
	err := kp.Query("exits", keyCondition, expressionAttributeValues, &exitsData)
	if err != nil {
		Logger.Error("Error querying exits", "room_id", roomID, "error", err)
		return nil, fmt.Errorf("error querying exits: %w", err)
	}

	// Process each exit data entry
	for _, exitData := range exitsData {
		exitID, err := uuid.Parse(exitData.ExitID)
		if err != nil {
			Logger.Error("Invalid exit UUID", "exit_id", exitData.ExitID, "error", err)
			continue
		}
		exits[exitData.Direction] = &Exit{
			ExitID:     exitID,
			Direction:  exitData.Direction,
			TargetRoom: nil, // This will be resolved later when all rooms are loaded
			Visible:    exitData.Visible,
		}
	}

	Logger.Info("Loaded exits for room", "room_id", roomID, "exit_count", len(exits))
	return exits, nil
}

// DisplayRooms logs information about all rooms, useful for debugging.
func DisplayRooms(rooms map[int64]*Room) {
	Logger.Info("Displaying rooms")
	for _, room := range rooms {
		Logger.Info("Room", "room_id", room.RoomID, "title", room.Title)
		for _, exit := range room.Exits {
			Logger.Info("  Exit", "direction", exit.Direction, "target_room", exit.TargetRoom)
		}
	}
}

// WriteRoom stores a single room into the DynamoDB database.
func (kp *KeyPair) WriteRoom(room *Room) error {
	if room == nil {
		return fmt.Errorf("cannot write nil room")
	}

	roomData := room.ToData()
	err := kp.Put("rooms", roomData)
	if err != nil {
		Logger.Error("Error writing room data", "room_id", room.RoomID, "error", err)
		return fmt.Errorf("error writing room data: %w", err)
	}

	// Write exits separately
	for _, exit := range room.Exits {
		if exit == nil {
			Logger.Warn("Skipping nil exit for room", "room_id", room.RoomID)
			continue
		}
		if exit.TargetRoom == nil {
			Logger.Warn("Skipping exit with nil TargetRoom for room", "room_id", room.RoomID, "direction", exit.Direction)
			continue
		}
		exitData := ExitData{
			ExitID:     exit.ExitID.String(),
			Direction:  exit.Direction,
			TargetRoom: exit.TargetRoom.RoomID,
			Visible:    exit.Visible,
		}
		err := kp.Put("exits", exitData)
		if err != nil {
			Logger.Error("Error writing exit data", "room_id", room.RoomID, "direction", exit.Direction, "error", err)
			return fmt.Errorf("error writing exit data: %w", err)
		}
	}

	Logger.Info("Successfully wrote room and exits to database", "room_id", room.RoomID)
	return nil
}

// SaveActiveRooms saves all active rooms to the database.
func SaveActiveRooms(s *Server) error {
	if s == nil {
		return fmt.Errorf("server is nil")
	}

	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	for roomID, room := range s.Rooms {
		if room == nil {
			Logger.Warn("Skipping nil room", "room_id", roomID)
			continue
		}
		if err := s.Database.WriteRoom(room); err != nil {
			Logger.Error("Error saving room", "room_id", roomID, "error", err)
			// Continue saving other rooms even if one fails
		}
	}

	Logger.Info("Finished saving active rooms")
	return nil
}

// NewRoom creates a new Room instance with initialized fields.
func NewRoom(roomID int64, area string, title string, description string) *Room {
	room := &Room{
		RoomID:      roomID,
		Area:        area,
		Title:       title,
		Description: description,
		Exits:       make(map[string]*Exit),
		Characters:  make(map[uuid.UUID]*Character),
		Items:       make(map[uuid.UUID]*Item),
		Mutex:       sync.Mutex{},
	}
	Logger.Info("Created room", "room_title", room.Title, "room_id", room.RoomID)
	return room
}

// AddExit adds an exit to the room's exits map.
func (r *Room) AddExit(exit *Exit) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if exit == nil {
		Logger.Warn("Attempted to add nil exit to room", "room_id", r.RoomID)
		return
	}

	r.Exits[exit.Direction] = exit
	Logger.Info("Added exit to room", "room_id", r.RoomID, "direction", exit.Direction)
}

// Move handles character movement from one room to another based on the direction.
func Move(c *Character, direction string) {
	Logger.Info("Player is attempting to move", "player_name", c.Name, "direction", direction)

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if c.Room == nil {
		c.Player.ToPlayer <- "\n\rYou are not in any room to move from.\n\r"
		Logger.Warn("Character has no current room", "character_name", c.Name)
		return
	}

	selectedExit, exists := c.Room.Exits[direction]
	if !exists {
		c.Player.ToPlayer <- "\n\rYou cannot go that way.\n\r"
		Logger.Warn("Invalid direction for movement", "character_name", c.Name, "direction", direction)
		return
	}

	if selectedExit.TargetRoom == nil {
		c.Player.ToPlayer <- "\n\rThe path leads nowhere.\n\r"
		Logger.Warn("Target room is nil", "character_name", c.Name, "direction", direction)
		return
	}

	newRoom := selectedExit.TargetRoom

	// Safely remove the character from the old room
	oldRoom := c.Room
	oldRoom.Mutex.Lock()
	delete(oldRoom.Characters, c.ID)
	oldRoom.Mutex.Unlock()
	SendRoomMessage(oldRoom, fmt.Sprintf("\n\r%s has left going %s.\n\r", c.Name, direction))

	// Update character's room
	c.Room = newRoom

	// Safely add the character to the new room
	newRoom.Mutex.Lock()
	if newRoom.Characters == nil {
		newRoom.Characters = make(map[uuid.UUID]*Character)
	}
	SendRoomMessage(newRoom, fmt.Sprintf("\n\r%s has arrived.\n\r", c.Name))
	newRoom.Characters[c.ID] = c
	newRoom.Mutex.Unlock()

	// Let the character look around the new room
	ExecuteLookCommand(c, []string{})
	Logger.Info("Character moved successfully", "character_name", c.Name, "new_room_id", newRoom.RoomID)
}

// SendRoomMessage sends a message to all characters in the room.
func SendRoomMessage(r *Room, message string) {
	Logger.Info("Sending message to room", "room_id", r.RoomID, "message", message)

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for _, character := range r.Characters {
		character.Player.ToPlayer <- message
		character.Player.ToPlayer <- character.Player.Prompt
	}
}

// RoomInfo generates a description of the room, including exits, characters, and items.
func RoomInfo(r *Room, character *Character) string {
	if r == nil {
		Logger.Error("Attempted to get room info for nil room", "character_name", character.Name)
		return "\n\rError: You are not in a valid room.\n\r"
	}
	if character == nil {
		Logger.Error("Attempted to get room info for nil character", "room_id", r.RoomID)
		return "\n\rError: Invalid character.\n\r"
	}

	var roomInfo strings.Builder

	// Room Title and Description
	roomInfo.WriteString(ApplyColor("bright_white", fmt.Sprintf("\n\r[%s]\n\r", r.Title)) + fmt.Sprintf("%s\n\r", r.Description))

	// Exits
	exits := sortedExits(r)
	if len(exits) == 0 {
		roomInfo.WriteString("There are no exits.\n\r")
	} else {
		roomInfo.WriteString("Obvious exits: ")
		roomInfo.WriteString(strings.Join(exits, ", "))
		roomInfo.WriteString("\n\r")
	}

	// Characters in the room
	otherCharacters := getOtherCharacters(r, character)
	if len(otherCharacters) > 0 {
		roomInfo.WriteString("Also here: ")
		roomInfo.WriteString(strings.Join(otherCharacters, ", "))
		roomInfo.WriteString("\n\r")
	} else {
		roomInfo.WriteString("You are alone.\n\r")
	}

	// Items in the room
	items := getVisibleItems(r)
	if len(items) > 0 {
		roomInfo.WriteString("Items in the room:\n\r")
		for _, item := range items {
			roomInfo.WriteString(fmt.Sprintf("- %s\n\r", item))
		}
	}

	return roomInfo.String()
}

// sortedExits returns a sorted list of exit directions from the room.
func sortedExits(r *Room) []string {
	Logger.Info("Sorting exits for room", "room_id", r.RoomID)

	if r.Exits == nil {
		Logger.Info("Exits map is nil for room", "room_id", r.RoomID)
		return []string{}
	}

	exits := make([]string, 0, len(r.Exits))
	for direction := range r.Exits {
		exits = append(exits, direction)
	}
	sort.Strings(exits)
	return exits
}

// ToData converts a Room to RoomData for database storage.
func (r *Room) ToData() *RoomData {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	exitIDs := make([]string, 0, len(r.Exits))
	for direction := range r.Exits {
		exitIDs = append(exitIDs, direction)
	}

	itemIDs := make([]string, 0, len(r.Items))
	for itemID := range r.Items {
		itemIDs = append(itemIDs, itemID.String())
	}

	return &RoomData{
		RoomID:      r.RoomID,
		Area:        r.Area,
		Title:       r.Title,
		Description: r.Description,
		ExitIDs:     exitIDs,
		ItemIDs:     itemIDs,
	}
}

// FromData populates a Room from RoomData.
func (r *Room) FromData(data *RoomData, exits map[string]*Exit, items map[string]*Item) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.RoomID = data.RoomID
	r.Area = data.Area
	r.Title = data.Title
	r.Description = data.Description

	r.Exits = make(map[string]*Exit)
	for _, direction := range data.ExitIDs {
		if exit, ok := exits[direction]; ok {
			r.Exits[direction] = exit
		}
	}

	r.Items = make(map[uuid.UUID]*Item)
	for _, itemIDStr := range data.ItemIDs {
		if itemID, err := uuid.Parse(itemIDStr); err == nil {
			if item, ok := items[itemIDStr]; ok {
				r.Items[itemID] = item
			}
		}
	}
}

// LoadAllExits loads all exits for all rooms.
func (kp *KeyPair) LoadAllExits() (map[int64]map[string]*Exit, error) {
	var exitsData []ExitData

	err := kp.Scan("exits", &exitsData)
	if err != nil {
		Logger.Error("Error scanning exits", "error", err)
		return nil, fmt.Errorf("error scanning exits: %w", err)
	}

	exits := make(map[int64]map[string]*Exit)
	for _, exitData := range exitsData {
		if _, exists := exits[exitData.RoomID]; !exists {
			exits[exitData.RoomID] = make(map[string]*Exit)
		}

		exitID, err := uuid.Parse(exitData.ExitID)
		if err != nil {
			Logger.Error("Invalid exit UUID", "exit_id", exitData.ExitID, "error", err)
			continue
		}

		exits[exitData.RoomID][exitData.Direction] = &Exit{
			ExitID:     exitID,
			Direction:  exitData.Direction,
			TargetRoom: nil, // This will be resolved later when linking rooms
			Visible:    exitData.Visible,
		}
	}

	Logger.Info("Loaded all exits", "total_rooms", len(exits))
	return exits, nil
}

// LoadItemsForRoom loads all items for a specific room
func (kp *KeyPair) LoadItemsForRoom(roomID int64) (map[uuid.UUID]*Item, error) {
	items := make(map[uuid.UUID]*Item)

	var itemsData []ItemData
	// Assume we have a way to query items by room ID
	err := kp.Query("items", "RoomID = :roomID", map[string]*dynamodb.AttributeValue{
		":roomID": {N: aws.String(strconv.FormatInt(roomID, 10))},
	}, &itemsData)

	if err != nil {
		return nil, fmt.Errorf("error querying items for room %d: %w", roomID, err)
	}

	for _, itemData := range itemsData {
		item, err := kp.itemFromData(&itemData)
		if err != nil {
			Logger.Error("Error creating item from data", "item_id", itemData.ItemID, "error", err)
			continue
		}
		items[item.ID] = item
	}

	return items, nil
}
