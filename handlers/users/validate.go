package userHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"
)

func ValidateUsernameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.ValidateUsernameRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Println("Username: ", req.Username)

	response, err := validateUsername(req)
	if err != nil {
		http.Error(w, "Error validating username", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func validateUsername(req types.ValidateUsernameRequest) (types.ValidateUsernameResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)

	var userResult types.User
	err := collection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&userResult)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return types.ValidateUsernameResponse{
				UsernameAvailable: true,
			}, nil
		}

		fmt.Println("Error finding user in the database:", err)
		return types.ValidateUsernameResponse{
			UsernameAvailable: false,
		}, err
	}

	return types.ValidateUsernameResponse{
		UsernameAvailable: false,
	}, nil
}
