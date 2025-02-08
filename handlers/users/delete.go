package userHandlers

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

func DeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("user")
	if username == "" {
		http.Error(w, "Missing required param \"user\"", http.StatusBadRequest)
		return
	}

	response, err := deleteAccount(username)
	if err != nil {
		http.Error(w, "Error deleting account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteAccount(username string) (types.DeleteAccountResponse, error) {
	// first go through and delete all journal entries associated with the account
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	entryCollection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	_, err := entryCollection.DeleteMany(ctx, bson.M{"username": username})
	if err != nil {
		fmt.Println("Error deleting all entries from the database: ", err)
		return types.DeleteAccountResponse{
			Success: false,
		}, err
	}

	// then delete the account
	userCollection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)
	_, err = userCollection.DeleteOne(ctx, bson.M{"username": username})
	if err != nil {
		fmt.Println("Error deleting the user from the database: ", err)
		return types.DeleteAccountResponse{
			Success: false,
		}, err
	}

	return types.DeleteAccountResponse{
		Success: true,
	}, nil
}
