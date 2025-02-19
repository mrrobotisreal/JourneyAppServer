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

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.UpdateUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := updateUser(req)
	if err != nil {
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func updateUser(req types.UpdateUserRequest) (types.UpdateUserResponse, error) {
	update := bson.M{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	salt, err := utils.GenerateSalt(10)
	if err != nil {
		fmt.Println("Error generating salt...", err)
		return types.UpdateUserResponse{
			Success: false,
		}, err
	}
	fmt.Println("Salt:", salt)
	update["salt"] = salt

	hashedPassword, err := utils.HashPassword(req.Password + salt)
	if err != nil {
		fmt.Println("Error hashing password+salt:", err)

		return types.UpdateUserResponse{
			Success: false,
		}, err
	}
	fmt.Println("Hashed password:", hashedPassword)
	update["password"] = hashedPassword

	collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)

	//apiKey, err := utils.GenerateSecureAPIKey()
	//if err != nil {
	//	fmt.Println("Error generating secure API key: ", err)
	//	return types.UpdateUserResponse{
	//		Success: false,
	//	}, err
	//}
	//update["apiKey"] = *apiKey
	//
	token, err := utils.GenerateAndStoreJWT(req.Username, req.SessionOption)
	if err != nil {
		fmt.Println("Error generating JWT: ", err)
		return types.UpdateUserResponse{
			Success: false,
		}, err
	}

	var user types.User
	err = collection.FindOneAndUpdate(ctx, bson.M{"username": req.Username}, bson.M{"$set": update}).Decode(&user)
	if err != nil {
		fmt.Println("Error attempting to update user: ", err)
		return types.UpdateUserResponse{
			Success: false,
		}, err
	}

	return types.UpdateUserResponse{
		Success: true,
		Token:   token,
		APIKey:  user.APIKey.Key,
	}, nil
}
