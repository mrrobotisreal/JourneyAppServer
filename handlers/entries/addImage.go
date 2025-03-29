package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"net/http"
	"strconv"
)

func AddImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.AddImageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := addImage(req, r)
	if err != nil {
		http.Error(w, "Error adding the image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func addImage(req types.AddImageRequest, r *http.Request) (types.AddImageResponse, error) {
	tx, err := db.SDB.Begin()
	if err != nil {
		utils.LM.Logger.Printf("Error starting transaction for image addition: entry=%s, error=%v", req.EntryID, err)
		return types.AddImageResponse{Success: false}, err
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
		return types.AddImageResponse{Success: false}, err
	}
	if !exists {
		utils.LM.Logger.Printf("Entry not found for image addition: entry=%s, userId=%s, timestamp=%v", req.EntryID, req.UserID, req.Timestamp)
		return types.AddImageResponse{Success: false}, nil
	}

	deleteQuery := `
        DELETE FROM entry_images 
        WHERE entry_id = ?
    `
	_, err = tx.Exec(deleteQuery, req.EntryID)
	if err != nil {
		utils.LM.Logger.Printf("Error deleting existing images for entry %s: %v", req.EntryID, err)
		return types.AddImageResponse{Success: false}, err
	}

	if len(req.Images) > 0 {
		insertQuery := `
            INSERT INTO entry_images (entry_id, image_url)
            VALUES (?, ?)
        `
		for _, img := range req.Images {
			_, err = tx.Exec(insertQuery, req.EntryID, img)
			if err != nil {
				utils.LM.Logger.Printf("Error inserting new image for entry %s: url=%s, error=%v", req.EntryID, img, err)
				return types.AddImageResponse{Success: false}, err
			}
		}
	}

	updateQuery := `
        UPDATE entries 
        SET last_updated = NOW() 
        WHERE entry_id = ? AND user_id = ? AND timestamp = ?
    `
	_, err = tx.Exec(updateQuery, req.EntryID, req.UserID, req.Timestamp)
	if err != nil {
		utils.LM.Logger.Printf("Error updating last_updated for entry %s: %v", req.EntryID, err)
		return types.AddImageResponse{Success: false}, err
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
			"image_count":  strconv.Itoa(len(req.Images)),
		}
		metadataJSON, _ := json.Marshal(metadata)
		analyticsQuery := `
            INSERT INTO analytics_events (
                user_id, event_type, object_type, object_id, event_time, metadata
            ) VALUES (?, ?, ?, ?, NOW(), ?)
        `
		_, err := db.SDB.Exec(analyticsQuery, req.UserID, "add_image", "entry", req.EntryID, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error for image addition to entry %s: %v", req.EntryID, err)
		}
	}()

	utils.LM.Logger.Printf("Successfully updated images for entry %s (new count: %d) for user %s", req.EntryID, len(req.Images), req.UserID)
	return types.AddImageResponse{Success: true}, nil

	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//
	//var entry types.Entry
	//err := collection.FindOneAndUpdate(ctx, bson.M{"id": req.EntryID, "userId": req.UserID, "timestamp": req.Timestamp}, bson.M{"$set": bson.M{"images": req.Images}}).Decode(&entry)
	//if err != nil {
	//	fmt.Println("Error adding the image to the entry in the database:", err)
	//	return types.AddImageResponse{
	//		Success: false,
	//	}, err
	//}
	//fmt.Println("Add image result is:", entry)
	//
	//return types.AddImageResponse{
	//	Success: true,
	//}, nil
}
