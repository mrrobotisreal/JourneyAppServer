package userHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("user")
	if username == "" {
		http.Error(w, "Missing required query param \"user\"", http.StatusBadRequest)
		return
	}

	response, err := getUser(username)
	if err != nil {
		http.Error(w, "Error fetching user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getUser(username string) (types.User, error) {
	var userResult types.User
	query := `
        SELECT user_id, username, password, salt, 
               api_key, api_key_created, api_key_last_used, api_key_expires_at, font 
        FROM users WHERE username = ?
    `
	err := db.SDB.QueryRow(query, username).Scan(
		&userResult.UserID, &userResult.Username, &userResult.Password, &userResult.Salt,
		&userResult.APIKey.Key, &userResult.APIKey.Created, &userResult.APIKey.LastUsed,
		&userResult.APIKey.ExpiresAt, &userResult.Font,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.LM.Logger.Printf("User not found in database: username=%s", username)
			return types.User{}, fmt.Errorf("user not found: %s", username)
		}
		utils.LM.Logger.Printf("Error querying user from database: username=%s, error=%v", username, err)
		return types.User{}, err
	}

	utils.LM.Logger.Printf("Successfully retrieved user: username=%s", username)
	return userResult, nil

	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)
	//
	//var userResult types.User
	//err := collection.FindOne(ctx, bson.M{"username": username}).Decode(&userResult)
	//if err != nil {
	//	fmt.Println("Error getting user from database: ", err)
	//	return types.User{}, err
	//}
	//
	//return userResult, nil
}
