package userHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Println("Username: ", req.Username)
	fmt.Println("Password: ", req.Password)

	response, err := createUser(req)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createUser(req types.CreateUserRequest) (types.CreateUserResponse, error) {
	salt, err := utils.GenerateSalt(10)
	if err != nil {
		fmt.Println("Error generating salt...", err)
		return types.CreateUserResponse{
			Success: false,
		}, err
	}
	fmt.Println("Salt:", salt)

	hashedPassword, err := utils.HashPassword(req.Password + salt)
	if err != nil {
		fmt.Println("Error hashing password+salt:", err)
		return types.CreateUserResponse{
			Success: false,
		}, err
	}
	fmt.Println("Hashed password:", hashedPassword)

	newUser := types.User{
		Username: req.Username,
		Password: hashedPassword,
		Salt:     salt,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)

	result, err := collection.InsertOne(ctx, newUser)
	if err != nil {
		fmt.Println("Error inserting new user into the database:", err)
		return types.CreateUserResponse{
			Success: false,
		}, err
	}
	fmt.Println("Result:", result)

	return types.CreateUserResponse{
		Success: true,
	}, nil
}
