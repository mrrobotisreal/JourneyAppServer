package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func AddImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.AddImageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := addImage(req)
	if err != nil {
		http.Error(w, "Error adding the image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func addImage(req types.AddImageRequest) (types.AddImageResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	var entry types.Entry
	err := collection.FindOneAndUpdate(ctx, bson.M{"id": req.EntryID, "userId": req.UserID, "timestamp": req.Timestamp}, bson.M{"$set": bson.M{"images": req.Images}}).Decode(&entry)
	if err != nil {
		fmt.Println("Error adding the image to the entry in the database:", err)
		return types.AddImageResponse{
			Success: false,
		}, err
	}
	fmt.Println("Add image result is:", entry)

	return types.AddImageResponse{
		Success: true,
	}, nil
}
