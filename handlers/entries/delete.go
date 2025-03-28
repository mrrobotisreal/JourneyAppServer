package entriesHandlers

import (
	"JourneyAppServer/aws"
	"JourneyAppServer/db"
	"JourneyAppServer/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DeleteEntryRequest struct {
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
}

func DeleteEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing required param \"id\".", http.StatusBadRequest)
		return
	}
	var req DeleteEntryRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	success, err := deleteEntry(id, req.UserID, req.Timestamp, r)
	if err != nil {
		http.Error(w, "Error deleting the entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": %v}`, success)
}

func deleteEntry(id, userId string, timestamp time.Time, r *http.Request) (bool, error) {
	tx, err := db.SDB.Begin()
	if err != nil {
		utils.LM.Logger.Printf("Error starting transaction for entry deletion: id=%s, userId=%s, error=%v", id, userId, err)
		return false, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	imgQuery := `
        SELECT image_url 
        FROM entry_images 
        WHERE entry_id = ?
    `
	imgRows, err := tx.Query(imgQuery, id)
	if err != nil {
		utils.LM.Logger.Printf("Error querying images for entry %s: %v", id, err)
		return false, err
	}
	defer imgRows.Close()

	var imageKeys []string
	for imgRows.Next() {
		var imageKey string
		if err := imgRows.Scan(&imageKey); err != nil {
			utils.LM.Logger.Printf("Error scanning image key for entry %s: %v", id, err)
			return false, err
		}
		imageKeys = append(imageKeys, imageKey)
	}
	if err := imgRows.Err(); err != nil {
		utils.LM.Logger.Printf("Image row iteration error for entry %s: %v", id, err)
		return false, err
	}

	for _, imageKey := range imageKeys {
		if res := aws.DeleteImage(imageKey); !res.Success {
			utils.LM.Logger.Printf("Error deleting S3 image %s for entry %s", imageKey, id)
		} else {
			utils.LM.Logger.Printf("Successfully deleted S3 image %s for entry %s", imageKey, id)
		}
	}

	deleteQuery := `
        DELETE FROM entries 
        WHERE entry_id = ? AND user_id = ? AND timestamp = ?
    `
	result, err := tx.Exec(deleteQuery, id, userId, timestamp)
	if err != nil {
		utils.LM.Logger.Printf("Error deleting entry from database: id=%s, userId=%s, timestamp=%v, error=%v", id, userId, timestamp, err)
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		utils.LM.Logger.Printf("Error checking rows affected for entry deletion: id=%s, userId=%s, error=%v", id, userId, err)
		return false, err
	}
	if rowsAffected == 0 {
		utils.LM.Logger.Printf("No entry found to delete: id=%s, userId=%s, timestamp=%v", id, userId, timestamp)
		return false, nil
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
		_, err := db.SDB.Exec(analyticsQuery, userId, "delete_entry", "entry", id, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error for entry deletion %s: %v", id, err)
		}
	}()

	utils.LM.Logger.Printf("Successfully deleted entry: id=%s, userId=%s, timestamp=%v", id, userId, timestamp)
	return true, nil

	////var deleteImagesFromAWSResult = aws.BulkDeleteImages()
	////if !deleteImagesFromAWSResult.Success {
	////	fmt.Println("Error deleting the image from AWS")
	////	return false, nil
	////}
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//
	//result, err := collection.DeleteOne(ctx, bson.M{"id": id, "userId": userId, "timestamp": timestamp})
	//if err != nil {
	//	fmt.Println("Error deleting the entry from the database:", err)
	//	return false, err
	//}
	//fmt.Println("Delete result is:", result)
	//
	//return true, nil
}
