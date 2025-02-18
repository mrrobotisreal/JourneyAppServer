package entriesHandlers

import (
	"JourneyAppServer/db"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type DeleteEntryRequest struct {
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
}

func DeleteEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing required param \"id\".", http.StatusBadRequest)
		return
	}
	var req DeleteEntryRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	success, err := deleteEntry(id, req.UserID, req.Timestamp)
	if err != nil {
		http.Error(w, "Error deleting the entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": %v}`, success)
}

func deleteEntry(id, userId string, timestamp time.Time) (bool, error) {
	//var deleteImagesFromAWSResult = aws.BulkDeleteImages()
	//if !deleteImagesFromAWSResult.Success {
	//	fmt.Println("Error deleting the image from AWS")
	//	return false, nil
	//}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	result, err := collection.DeleteOne(ctx, bson.M{"id": id, "userId": userId, "timestamp": timestamp})
	if err != nil {
		fmt.Println("Error deleting the entry from the database:", err)
		return false, err
	}
	fmt.Println("Delete result is:", result)

	return true, nil
}
