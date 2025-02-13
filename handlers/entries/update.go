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

func UpdateEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.UpdateEntryRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	//if req.UserID == "" {
	//	http.Error(w, "Missing required body property \"userId\"", http.StatusBadRequest)
	//	return
	//}
	if req.Timestamp.IsZero() {
		http.Error(w, "Missing required body property \"timestamp\"", http.StatusBadRequest)
		return
	}

	response, err := updateEntry(req)
	if err != nil {
		http.Error(w, "Error updating the entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func updateEntry(req types.UpdateEntryRequest) (types.UpdateEntryResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{}

	if req.Text != "" {
		update["text"] = req.Text
	}

	if req.Locations != nil && len(req.Locations) > 0 {
		update["locations"] = req.Locations
	}

	if req.Tags != nil && len(req.Tags) > 0 {
		update["tags"] = req.Tags
	}

	if req.Images != nil && len(req.Images) > 0 {
		update["images"] = req.Images
	}

	if req.LastUpdated.IsZero() {
		update["lastUpdated"] = time.Now().UTC()
	}

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	var entry types.Entry
	err := collection.FindOneAndUpdate(ctx, bson.M{"id": req.ID, "userId": req.UserID, "timestamp": req.Timestamp}, bson.M{"$set": update}).Decode(&entry)
	if err != nil {
		fmt.Println("Error finding and updating the entry in the database:", err)
		return types.UpdateEntryResponse{
			Success: false,
		}, err
	}

	return types.UpdateEntryResponse{
		Success: true,
	}, nil
}
