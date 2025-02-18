package entriesHandlers

import (
	"JourneyAppServer/aws"
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func DeleteImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.DeleteImageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := deleteImage(req)
	if err != nil {
		http.Error(w, "Error deleting the image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteImage(req types.DeleteImageRequest) (types.DeleteImageResponse, error) {
	var deleteImageFromAWSResult = aws.DeleteImage(req.ImageToDelete)
	if !deleteImageFromAWSResult.Success {
		fmt.Println("Error deleting the image from AWS")
		return types.DeleteImageResponse{
			Success: false,
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	var entry types.Entry
	err := collection.FindOneAndUpdate(ctx, bson.M{"id": req.EntryID, "userId": req.UserID, "timestamp": req.Timestamp}, bson.M{"$set": bson.M{"images": req.Images}}).Decode(&entry)
	if err != nil {
		fmt.Println("Error deleting the image from the entry in the database:", err)
		return types.DeleteImageResponse{
			Success: false,
		}, err
	}
	fmt.Println("Delete image result is:", entry)

	return types.DeleteImageResponse{
		Success: true,
	}, nil
}
