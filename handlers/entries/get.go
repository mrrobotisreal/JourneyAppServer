package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"time"
)

func GetEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	userId := r.URL.Query().Get("userId")
	timestampStr := r.URL.Query().Get("timestamp")
	if id == "" {
		http.Error(w, "Missing required param \"id\"", http.StatusBadRequest)
		return
	}
	if userId == "" {
		http.Error(w, "Missing required param \"userId\"", http.StatusBadRequest)
		return
	}
	if timestampStr == "" {
		http.Error(w, "Missing required param \"timestamp\"", http.StatusBadRequest)
		return
	}
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		http.Error(w, "Invalid timestamp format", http.StatusBadRequest)
		return
	}

	response, err := getEntry(id, userId, timestamp)
	if err != nil {
		http.Error(w, "Error getting the entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getEntry(id, userId string, timestamp time.Time) (types.Entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	var entry types.Entry
	err := collection.FindOne(ctx, bson.M{"id": id, "userId": userId, "timestamp": timestamp}).Decode(&entry)
	if err != nil {
		fmt.Println("Error finding the entry in the database: ", err)
		return types.Entry{}, err
	}

	return entry, nil
}
