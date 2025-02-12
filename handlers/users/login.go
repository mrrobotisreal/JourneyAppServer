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

	if !utils.IsValidSessionOption(req.SessionOption) {
		http.Error(w, "Invalid session option", http.StatusBadRequest)
		return
	}

	fmt.Println("Incoming login request:", req.Username) // TODO: start maintaining logs of login requests when failed

	response, err := login(req)
	if err != nil {
		http.Error(w, "Error logging in", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func login(req types.LoginRequest) (types.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	if !isPasswordValid {
		fmt.Println("INVALID PASSWORD ATTEMPTED!") // TODO: add logs for this
		return types.LoginResponse{
			Success: false,
		}, nil
	}

	token, err := utils.GenerateAndStoreJWT(req.Username, req.SessionOption)
	if err != nil {
		fmt.Println("Error generating token: ", err)
		return types.LoginResponse{
			Success: false,
		}, err
	}

	//shouldRespondWithAPIKey := false
	APIKey := ""

	if utils.IsKeyRotationNeeded(&userResult.APIKey) {
		newAPIKey, err := utils.GenerateSecureAPIKey()
		if err != nil {
			fmt.Println("Error rotating the API key: ", err)
		} else {
			_, err = collection.UpdateOne(ctx, bson.M{"username": req.Username}, bson.M{"$set": bson.M{"apiKey": newAPIKey}})
			if err != nil {
				fmt.Println("Error updating rotated API key: ", err)
			} else {
				APIKey = newAPIKey.Key
				//shouldRespondWithAPIKey = true
			}
		}
	} else {
		APIKey = userResult.APIKey.Key
	}

	//if shouldRespondWithAPIKey {
	//	return types.LoginResponse{
	//		Success: true,
	//		Token:   token,
	//		APIKey:  APIKey,
	//	}, nil
	//} else if req.RespondWithAPIKey {
	//	if req.Key == os.Getenv("RESPOND_WITH_API_KEY_KEY") {
	//		return types.LoginResponse{
	//			Success: true,
	//			Token:   token,
	//			APIKey:  userResult.APIKey.Key,
	//		}, nil
	//	}
	//}

	return types.LoginResponse{
		Success: true,
		Token:   token,
		APIKey:  APIKey,
	}, nil
}
