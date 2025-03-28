package userHandlers

import (
	"JourneyAppServer/aws"
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"net/http"
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

	response, err := deleteAccount(username, r)
	if err != nil {
		http.Error(w, "Error deleting account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteAccount(username string, r *http.Request) (types.DeleteAccountResponse, error) {
	query := `
        SELECT ei.image_url
        FROM entries e
        JOIN entry_images ei ON e.entry_id = ei.entry_id
        WHERE e.username = ?
    `
	rows, err := db.SDB.Query(query, username)
	if err != nil {
		utils.LM.Logger.Printf("Error querying entry images for user %s: %v", username, err)
		return types.DeleteAccountResponse{Success: false}, err
	}
	defer rows.Close()

	var imageKeys []string
	for rows.Next() {
		var imageKey string
		if err := rows.Scan(&imageKey); err != nil {
			utils.LM.Logger.Printf("Error scanning image key for user %s: %v", username, err)
			return types.DeleteAccountResponse{Success: false}, err
		}
		imageKeys = append(imageKeys, imageKey)
	}
	if err := rows.Err(); err != nil {
		utils.LM.Logger.Printf("Row iteration error for user %s: %v", username, err)
		return types.DeleteAccountResponse{Success: false}, err
	}

	for _, imageKey := range imageKeys {
		if res := aws.DeleteImage(imageKey); !res.Success {
			utils.LM.Logger.Printf("Error deleting S3 image %s for user %s", imageKey, username)
		} else {
			utils.LM.Logger.Printf("Successfully deleted S3 image %s for user %s", imageKey, username)
		}
	}

	deleteQuery := `DELETE FROM users WHERE username = ?`
	result, err := db.SDB.Exec(deleteQuery, username)
	if err != nil {
		utils.LM.Logger.Printf("Error deleting user %s from database: %v", username, err)
		return types.DeleteAccountResponse{Success: false}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		utils.LM.Logger.Printf("Error checking rows affected for user %s: %v", username, err)
		return types.DeleteAccountResponse{Success: false}, err
	}
	if rowsAffected == 0 {
		utils.LM.Logger.Printf("No user found to delete for username %s", username)
		return types.DeleteAccountResponse{Success: false}, nil
	}

	go func() {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		metadata := map[string]string{
			"source":       "api",
			"client_ip":    ip,
			"user_agent":   r.Header.Get("User-Agent"),
			"app_version":  r.Header.Get("X-App-Version"),
			"os_version":   r.Header.Get("X-OS-Version"),
			"device_model": r.Header.Get("X-Device-Model"),
		}
		metadataJSON, _ := json.Marshal(metadata)
		analyticsQuery := `
            INSERT INTO analytics_events (
                user_id, event_type, object_type, object_id, event_time, metadata
            ) VALUES (?, ?, ?, ?, NOW(), ?)
        `
		_, err := db.SDB.Exec(analyticsQuery, username, "delete account", "user", username, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error for delete account %s: %v", username, err)
		}
	}()

	utils.LM.Logger.Printf("Successfully deleted account for username %s", username)
	return types.DeleteAccountResponse{Success: true}, nil

	//// first go through and delete all journal entries associated with the account
	//ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	//defer cancel()
	//
	//userCollection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)
	//entryCollection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//
	//pipeline := mongo.Pipeline{}
	//
	//matchUser := bson.D{{Key: "$match", Value: bson.D{
	//	{Key: "username", Value: username},
	//}}}
	//pipeline = append(pipeline, matchUser)
	//
	//cursor, err := entryCollection.Aggregate(ctx, pipeline)
	//if err != nil {
	//	log.Println("Aggregate all entries error: ", err)
	//	return types.DeleteAccountResponse{
	//		Success: false,
	//	}, err
	//}
	//defer cursor.Close(ctx)
	//
	//for cursor.Next(ctx) {
	//	var EntryImages struct {
	//		IDM    string   `bson:"_id"`
	//		ID     string   `bson:"id"`
	//		Images []string `bson:"images"`
	//	}
	//	if err := cursor.Decode(&EntryImages); err != nil {
	//		log.Println("Error decoding individual entry:", err)
	//		return types.DeleteAccountResponse{
	//			Success: false,
	//		}, err
	//	}
	//
	//	if len(EntryImages.Images) > 0 {
	//		for _, imageKey := range EntryImages.Images {
	//			if res := aws.DeleteImage(imageKey); !res.Success {
	//				log.Println("Error deleting individual image: ", imageKey)
	//			} else {
	//				log.Println("Successfully deleted image: ", imageKey, "!")
	//			}
	//		}
	//	}
	//}
	//if err := cursor.Err(); err != nil {
	//	log.Println("Cursor encountered an error: ", err)
	//	return types.DeleteAccountResponse{
	//		Success: false,
	//	}, err
	//}
	//
	//_, err = entryCollection.DeleteMany(ctx, bson.M{"username": username})
	//if err != nil {
	//	fmt.Println("Error deleting all entries from the database: ", err)
	//	return types.DeleteAccountResponse{
	//		Success: false,
	//	}, err
	//}
	//
	//// then delete the account
	//_, err = userCollection.DeleteOne(ctx, bson.M{"username": username})
	//if err != nil {
	//	fmt.Println("Error deleting the user from the database: ", err)
	//	return types.DeleteAccountResponse{
	//		Success: false,
	//	}, err
	//}
	//
	//return types.DeleteAccountResponse{
	//	Success: true,
	//}, nil
}
