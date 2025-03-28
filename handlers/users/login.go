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

	response, err := login(req, r)
	if err != nil {
		http.Error(w, "Error logging in", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func login(req types.LoginRequest, r *http.Request) (types.LoginResponse, error) {
	var userResult types.User
	query := `
        SELECT user_id, username, password, salt, 
               api_key, api_key_created, api_key_last_used, api_key_expires_at, font 
        FROM users WHERE username = ?
    `
	err := db.SDB.QueryRow(query, req.Username).Scan(
		&userResult.UserID, &userResult.Username, &userResult.Password, &userResult.Salt,
		&userResult.APIKey.Key, &userResult.APIKey.Created, &userResult.APIKey.LastUsed,
		&userResult.APIKey.ExpiresAt, &userResult.Font,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.LM.Logger.Printf("User not found: username=%s", req.Username)
			return types.LoginResponse{Success: false}, nil
		}
		utils.LM.Logger.Printf("Database query error: %v", err)
		return types.LoginResponse{Success: false}, err
	}

	isPasswordValid := utils.CheckPasswordHash(req.Password+userResult.Salt, userResult.Password)
	if !isPasswordValid {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		utils.LM.Logger.Printf("Invalid password attempt: username=%s, client_ip=%s, user_agent=%s",
			req.Username, ip, r.Header.Get("User-Agent"))
		return types.LoginResponse{Success: false}, nil
	}

	token, err := utils.GenerateAndStoreJWT(req.Username, req.SessionOption)
	if err != nil {
		utils.LM.Logger.Printf("JWT generation error: %v", err)
		return types.LoginResponse{Success: false}, err
	}

	APIKey := userResult.APIKey.Key
	if utils.IsKeyRotationNeeded(&userResult.APIKey) {
		newAPIKey, err := utils.GenerateSecureAPIKey()
		if err != nil {
			utils.LM.Logger.Printf("API key rotation error: %v", err)
		} else {
			updateQuery := `
                UPDATE users 
                SET api_key = ?, api_key_created = ?, api_key_last_used = ?, api_key_expires_at = ?
                WHERE username = ?
            `
			result, err := db.SDB.Exec(updateQuery, newAPIKey.Key, newAPIKey.Created, newAPIKey.LastUsed, newAPIKey.ExpiresAt, req.Username)
			if err != nil {
				utils.LM.Logger.Printf("API key update error: %v", err)
			} else {
				rowsAffected, err := result.RowsAffected()
				if err == nil && rowsAffected == 1 {
					APIKey = newAPIKey.Key
				} else {
					utils.LM.Logger.Printf("Unexpected API key update result: err=%v, rows=%d", err, rowsAffected)
				}
			}
		}
	}

	go func() {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		metadata := map[string]string{
			"source":         "api",
			"client_ip":      ip,
			"user_agent":     r.Header.Get("User-Agent"),
			"app_version":    r.Header.Get("X-App-Version"),
			"os_version":     r.Header.Get("X-OS-Version"),
			"device_model":   r.Header.Get("X-Device-Model"),
			"session_option": req.SessionOption,
		}
		metadataJSON, _ := json.Marshal(metadata)
		analyticsQuery := `
            INSERT INTO analytics_events (
                user_id, event_type, object_type, object_id, event_time, meta_data
            ) VALUES (?, ?, ?, ?, NOW(), ?)
        `
		_, err := db.SDB.Exec(analyticsQuery, userResult.UserID, "login", "user", userResult.UserID, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error: %v", err)
		}
	}()

	return types.LoginResponse{
		UserID:   userResult.UserID,
		Username: userResult.Username,
		Success:  true,
		Token:    token,
		APIKey:   APIKey,
		Font:     userResult.Font,
	}, nil

	//
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)
	//
	//var userResult types.User
	//err := collection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&userResult)
	//if err != nil {
	//	fmt.Println("Error finding user in the database:", err)
	//	return types.LoginResponse{
	//		Success: false,
	//	}, err
	//}
	//
	//isPasswordValid := utils.CheckPasswordHash(req.Password+userResult.Salt, userResult.Password)
	//if !isPasswordValid {
	//	fmt.Println("INVALID PASSWORD ATTEMPTED!") // TODO: add logs for this
	//	return types.LoginResponse{
	//		Success: false,
	//	}, nil
	//}
	//
	//token, err := utils.GenerateAndStoreJWT(req.Username, req.SessionOption)
	//if err != nil {
	//	fmt.Println("Error generating token: ", err)
	//	return types.LoginResponse{
	//		Success: false,
	//	}, err
	//}
	//
	////shouldRespondWithAPIKey := false
	//APIKey := ""
	//
	//if utils.IsKeyRotationNeeded(&userResult.APIKey) {
	//	newAPIKey, err := utils.GenerateSecureAPIKey()
	//	if err != nil {
	//		fmt.Println("Error rotating the API key: ", err)
	//	} else {
	//		_, err = collection.UpdateOne(ctx, bson.M{"username": req.Username}, bson.M{"$set": bson.M{"apiKey": newAPIKey}})
	//		if err != nil {
	//			fmt.Println("Error updating rotated API key: ", err)
	//		} else {
	//			APIKey = newAPIKey.Key
	//			//shouldRespondWithAPIKey = true
	//		}
	//	}
	//} else {
	//	APIKey = userResult.APIKey.Key
	//}
	//
	////if shouldRespondWithAPIKey {
	////	return types.LoginResponse{
	////		Success: true,
	////		Token:   token,
	////		APIKey:  APIKey,
	////	}, nil
	////} else if req.RespondWithAPIKey {
	////	if req.Key == os.Getenv("RESPOND_WITH_API_KEY_KEY") {
	////		return types.LoginResponse{
	////			Success: true,
	////			Token:   token,
	////			APIKey:  userResult.APIKey.Key,
	////		}, nil
	////	}
	////}
	//
	//return types.LoginResponse{
	//	UserID:   userResult.UserID,
	//	Username: userResult.Username,
	//	Success:  true,
	//	Token:    token,
	//	APIKey:   APIKey,
	//	Font:     userResult.Font,
	//}, nil
}
