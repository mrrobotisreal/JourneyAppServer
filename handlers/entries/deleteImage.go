package entriesHandlers

import (
	"JourneyAppServer/aws"
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"net/http"
)

func DeleteImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.DeleteImageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := deleteImage(req, r)
	if err != nil {
		http.Error(w, "Error deleting the image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteImage(req types.DeleteImageRequest, r *http.Request) (types.DeleteImageResponse, error) {
	awsResult := aws.DeleteImage(req.ImageToDelete)
	if !awsResult.Success {
		utils.LM.Logger.Printf("Error deleting image %s from AWS for entry %s", req.ImageToDelete, req.EntryID)
		return types.DeleteImageResponse{Success: false}, nil
	}

	tx, err := db.SDB.Begin()
	if err != nil {
		utils.LM.Logger.Printf("Error starting transaction for image deletion: entry=%s, error=%v", req.EntryID, err)
		return types.DeleteImageResponse{Success: false}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	var exists bool
	checkQuery := `
        SELECT EXISTS(
            SELECT 1 FROM entries 
            WHERE entry_id = ? AND user_id = ? AND timestamp = ?
        )
    `
	err = tx.QueryRow(checkQuery, req.EntryID, req.UserID, req.Timestamp).Scan(&exists)
	if err != nil {
		utils.LM.Logger.Printf("Error checking entry existence: entry=%s, userId=%s, error=%v", req.EntryID, req.UserID, err)
		return types.DeleteImageResponse{Success: false}, err
	}
	if !exists {
		utils.LM.Logger.Printf("Entry not found for image deletion: entry=%s, userId=%s, timestamp=%v", req.EntryID, req.UserID, req.Timestamp)
		return types.DeleteImageResponse{Success: false}, nil
	}

	deleteQuery := `
        DELETE FROM entry_images 
        WHERE entry_id = ? AND image_url = ?
    `
	result, err := tx.Exec(deleteQuery, req.EntryID, req.ImageToDelete)
	if err != nil {
		utils.LM.Logger.Printf("Error deleting image %s from entry %s in database: %v", req.ImageToDelete, req.EntryID, err)
		return types.DeleteImageResponse{Success: false}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		utils.LM.Logger.Printf("Error checking rows affected for image deletion: entry=%s, error=%v", req.EntryID, err)
		return types.DeleteImageResponse{Success: false}, err
	}
	if rowsAffected == 0 {
		utils.LM.Logger.Printf("Image %s not found in entry %s", req.ImageToDelete, req.EntryID)
		return types.DeleteImageResponse{Success: false}, nil
	}

	updateQuery := `
        UPDATE entries 
        SET last_updated = NOW() 
        WHERE entry_id = ? AND user_id = ? AND timestamp = ?
    `
	_, err = tx.Exec(updateQuery, req.EntryID, req.UserID, req.Timestamp)
	if err != nil {
		utils.LM.Logger.Printf("Error updating last_updated for entry %s: %v", req.EntryID, err)
		return types.DeleteImageResponse{Success: false}, err
	}
	
	go func() {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		metadata := map[string]string{
			"source":        "api",
			"client_ip":     ip,
			"user_agent":    r.Header.Get("User-Agent"),
			"app_version":   r.Header.Get("X-App-Version"),
			"os_version":    r.Header.Get("X-OS-Version"),
			"device_model":  r.Header.Get("X-Device-Model"),
			"image_deleted": req.ImageToDelete,
		}
		metadataJSON, _ := json.Marshal(metadata)
		analyticsQuery := `
            INSERT INTO analytics_events (
                user_id, event_type, object_type, object_id, event_time, metadata
            ) VALUES (?, ?, ?, ?, NOW(), ?)
        `
		_, err := db.SDB.Exec(analyticsQuery, req.UserID, "delete_image", "entry", req.EntryID, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error for image deletion from entry %s: %v", req.EntryID, err)
		}
	}()

	utils.LM.Logger.Printf("Successfully deleted image %s from entry %s for user %s", req.ImageToDelete, req.EntryID, req.UserID)
	return types.DeleteImageResponse{Success: true}, nil

	//var deleteImageFromAWSResult = aws.DeleteImage(req.ImageToDelete)
	//if !deleteImageFromAWSResult.Success {
	//	fmt.Println("Error deleting the image from AWS")
	//	return types.DeleteImageResponse{
	//		Success: false,
	//	}, nil
	//}
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//
	//var entry types.Entry
	//err := collection.FindOneAndUpdate(ctx, bson.M{"id": req.EntryID, "userId": req.UserID, "timestamp": req.Timestamp}, bson.M{"$set": bson.M{"images": req.Images}}).Decode(&entry)
	//if err != nil {
	//	fmt.Println("Error deleting the image from the entry in the database:", err)
	//	return types.DeleteImageResponse{
	//		Success: false,
	//	}, err
	//}
	//fmt.Println("Delete image result is:", entry)
	//
	//return types.DeleteImageResponse{
	//	Success: true,
	//}, nil
}
