package userHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"time"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Println("Incoming login request:", req.Username)

	response, err := login(req)
	if err != nil {
		http.Error(w, "Error logging in", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func login(req types.LoginRequest) (types.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)

	var userResult types.User
	err := collection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&userResult)
	if err != nil {
		fmt.Println("Error finding user in the database:", err)
		return types.LoginResponse{
			Success: false,
		}, err
	}

	isPasswordValid := utils.CheckPasswordHash(req.Password+userResult.Salt, userResult.Password)

	return types.LoginResponse{
		Success: isPasswordValid,
	}, nil
}
