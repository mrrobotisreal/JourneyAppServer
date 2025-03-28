package userHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
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

	if !utils.IsValidSessionOption(req.SessionOption) {
		http.Error(w, "Invalid session option", http.StatusBadRequest)
		return
	}

	fmt.Println("Username: ", req.Username)
	fmt.Println("Password: ", req.Password)
	fmt.Println("SessionOption: ", req.SessionOption)

	response, err := createUser(req, r)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createUser(req types.CreateUserRequest, r *http.Request) (types.CreateUserResponse, error) {
	salt, err := utils.GenerateSalt(10)
	if err != nil {
		utils.LM.Logger.Printf("Generate salt error: %v", err)
		return types.CreateUserResponse{
			Success: false,
		}, err
	}
	fmt.Println("Salt:", salt)

	hashedPassword, err := utils.HashPassword(req.Password + salt)
	if err != nil {
		utils.LM.Logger.Printf("Hash password error: %v", err)
		return types.CreateUserResponse{
			Success: false,
		}, err
	}
	fmt.Println("Hashed password:", hashedPassword)

	apiKey, err := utils.GenerateSecureAPIKey()
	if err != nil {
		utils.LM.Logger.Printf("Generate secure API key error: %v", err)
		return types.CreateUserResponse{
			Success: false,
		}, err
	}

	token, err := utils.GenerateAndStoreJWT(req.Username, req.SessionOption)
	if err != nil {
		utils.LM.Logger.Printf("Generate and store JWT error: %v", err)
		return types.CreateUserResponse{
			Success: false,
		}, err
	}

	userId := uuid.New().String()

	query := `
        INSERT INTO users (
            user_id, username, password, salt, api_key, 
            api_key_created, api_key_last_used, api_key_expires_at, font
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
	result, err := db.SDB.Exec(query,
		userId, req.Username, hashedPassword, salt, apiKey.Key,
		apiKey.Created, apiKey.LastUsed, apiKey.ExpiresAt, "Default",
	)
	if err != nil {
		utils.LM.Logger.Printf("Error inserting new user into the database: %v", err)
		return types.CreateUserResponse{Success: false}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected != 1 {
		fmt.Println("Unexpected result from user insert:", err, rowsAffected)
		utils.LM.Logger.Printf("Unexpected result from user insert: result = %d; err = %v", rowsAffected, err)
		return types.CreateUserResponse{Success: false}, err
	}

	go func() {
		analyticsQuery := `
            INSERT INTO analytics_events (
                user_id, event_type, object_type, object_id, event_time, meta_data
            ) VALUES (?, ?, ?, ?, NOW(), ?)
        `
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
		_, err := db.SDB.Exec(analyticsQuery, userId, "create user", "user", userId, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error: %v", err)
		}
	}()

	return types.CreateUserResponse{
		UserID:   userId,
		Username: req.Username,
		Success:  true,
		Token:    token,
		APIKey:   apiKey.Key,
		Font:     "Default",
	}, nil

	//----------------------------Below is old MongoDB code---------------------------//

	//newUser := types.User{
	//	UserID:   userId,
	//	Username: req.Username,
	//	Password: hashedPassword,
	//	Salt:     salt,
	//	APIKey:   *apiKey,
	//	Font:     "Default",
	//}
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)
	//
	//_, err = collection.InsertOne(ctx, newUser)
	//if err != nil {
	//	fmt.Println("Error inserting new user into the database:", err)
	//	return types.CreateUserResponse{
	//		Success: false,
	//	}, err
	//}
	//
	//return types.CreateUserResponse{
	//	UserID:   userId,
	//	Username: req.Username,
	//	Success:  true,
	//	Token:    token,
	//	APIKey:   apiKey.Key,
	//	Font:     "Default",
	//}, nil
}
