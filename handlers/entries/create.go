package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"time"
)

func CreateNewEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.CreateNewEntryRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := createNewEntry(req)
	if err != nil {
		http.Error(w, "Error creating new entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createNewEntry(req types.CreateNewEntryRequest) (types.CreateNewEntryResponse, error) {
	newEntry := types.Entry{
		ID:        uuid.New().String(),
		Username:  req.Username,
		Text:      req.Text,
		Timestamp: req.Timestamp,
		Locations: req.Locations,
		Tags:      req.Tags,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	result, err := collection.InsertOne(ctx, newEntry)
	if err != nil {
		fmt.Println("Error inserting new entry into the database:", err)
		return types.CreateNewEntryResponse{}, err
	}
	fmt.Println("Result:", result)

	return types.CreateNewEntryResponse{
		UUID: newEntry.ID,
	}, nil
}
