package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"net/http"
	"time"
)

func UpdateEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.UpdateEntryRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	//if req.UserID == "" {
	//	http.Error(w, "Missing required body property \"userId\"", http.StatusBadRequest)
	//	return
	//}
	if req.Timestamp.IsZero() {
		http.Error(w, "Missing required body property \"timestamp\"", http.StatusBadRequest)
		return
	}

	response, err := updateEntry(req, r)
	if err != nil {
		http.Error(w, "Error updating the entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func updateEntry(req types.UpdateEntryRequest, r *http.Request) (types.UpdateEntryResponse, error) {
	tx, err := db.SDB.Begin()
	if err != nil {
		utils.LM.Logger.Printf("Error starting transaction for entry update: id=%s, userId=%s, error=%v", req.ID, req.UserID, err)
		return types.UpdateEntryResponse{Success: false}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM entries WHERE entry_id = ? AND user_id = ? AND timestamp = ?)`
	err = tx.QueryRow(checkQuery, req.ID, req.UserID, req.Timestamp).Scan(&exists)
	if err != nil {
		utils.LM.Logger.Printf("Error checking entry existence: id=%s, userId=%s, error=%v", req.ID, req.UserID, err)
		return types.UpdateEntryResponse{Success: false}, err
	}
	if !exists {
		utils.LM.Logger.Printf("Entry not found for update: id=%s, userId=%s, timestamp=%v", req.ID, req.UserID, req.Timestamp)
		return types.UpdateEntryResponse{Success: false}, nil
	}

	if req.Text != "" || !req.LastUpdated.IsZero() {
		updateQuery := `UPDATE entries SET `
		var args []interface{}
		if req.Text != "" {
			updateQuery += "text = ?, "
			args = append(args, req.Text)
		}
		lastUpdated := req.LastUpdated
		if lastUpdated.IsZero() {
			lastUpdated = time.Now().UTC()
		}
		updateQuery += "last_updated = ? "
		args = append(args, lastUpdated)
		updateQuery += "WHERE entry_id = ? AND user_id = ? AND timestamp = ?"
		args = append(args, req.ID, req.UserID, req.Timestamp)

		result, err := tx.Exec(updateQuery, args...)
		if err != nil {
			utils.LM.Logger.Printf("Error updating entry text/last_updated: id=%s, userId=%s, error=%v", req.ID, req.UserID, err)
			return types.UpdateEntryResponse{Success: false}, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected == 0 {
			utils.LM.Logger.Printf("No rows affected updating entry: id=%s, userId=%s, error=%v", req.ID, req.UserID, err)
			return types.UpdateEntryResponse{Success: false}, err
		}
	}

	if req.Locations != nil && len(req.Locations) > 0 {
		_, err = tx.Exec(`DELETE FROM entry_locations WHERE entry_id = ?`, req.ID)
		if err != nil {
			utils.LM.Logger.Printf("Error deleting old locations for entry %s: %v", req.ID, err)
			return types.UpdateEntryResponse{Success: false}, err
		}
		for _, loc := range req.Locations {
			locQuery := `
                INSERT INTO entry_locations (entry_id, latitude, longitude, display_name)
                VALUES (?, ?, ?, ?)
            `
			_, err = tx.Exec(locQuery, req.ID, loc.Latitude, loc.Longitude, loc.DisplayName)
			if err != nil {
				utils.LM.Logger.Printf("Error inserting location for entry %s: %v", req.ID, err)
				return types.UpdateEntryResponse{Success: false}, err
			}
		}
	}

	if req.Tags != nil && len(req.Tags) > 0 {
		_, err = tx.Exec(`DELETE FROM entry_tags WHERE entry_id = ?`, req.ID)
		if err != nil {
			utils.LM.Logger.Printf("Error deleting old tags for entry %s: %v", req.ID, err)
			return types.UpdateEntryResponse{Success: false}, err
		}
		for _, tag := range req.Tags {
			tagQuery := `
                INSERT INTO entry_tags (entry_id, tag_key, tag_value)
                VALUES (?, ?, ?)
            `
			_, err = tx.Exec(tagQuery, req.ID, tag.Key, tag.Value)
			if err != nil {
				utils.LM.Logger.Printf("Error inserting tag for entry %s: %v", req.ID, err)
				return types.UpdateEntryResponse{Success: false}, err
			}
		}
	}

	if req.Images != nil && len(req.Images) > 0 {
		_, err = tx.Exec(`DELETE FROM entry_images WHERE entry_id = ?`, req.ID)
		if err != nil {
			utils.LM.Logger.Printf("Error deleting old images for entry %s: %v", req.ID, err)
			return types.UpdateEntryResponse{Success: false}, err
		}
		for _, image := range req.Images {
			imgQuery := `
                INSERT INTO entry_images (entry_id, image_url)
                VALUES (?, ?)
            `
			_, err = tx.Exec(imgQuery, req.ID, image)
			if err != nil {
				utils.LM.Logger.Printf("Error inserting image for entry %s: %v", req.ID, err)
				return types.UpdateEntryResponse{Success: false}, err
			}
		}
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
		_, err := db.SDB.Exec(analyticsQuery, req.UserID, "update_entry", "entry", req.ID, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error for entry update %s: %v", req.ID, err)
		}
	}()

	utils.LM.Logger.Printf("Successfully updated entry: id=%s, userId=%s", req.ID, req.UserID)
	return types.UpdateEntryResponse{Success: true}, nil

	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//update := bson.M{}
	//
	//if req.Text != "" {
	//	update["text"] = req.Text
	//}
	//
	//if req.Locations != nil && len(req.Locations) > 0 {
	//	update["locations"] = req.Locations
	//}
	//
	//if req.Tags != nil && len(req.Tags) > 0 {
	//	update["tags"] = req.Tags
	//}
	//
	//if req.Images != nil && len(req.Images) > 0 {
	//	update["images"] = req.Images
	//}
	//
	//if req.LastUpdated.IsZero() {
	//	update["lastUpdated"] = time.Now().UTC()
	//}
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//
	//var entry types.Entry
	//err := collection.FindOneAndUpdate(ctx, bson.M{"id": req.ID, "userId": req.UserID, "timestamp": req.Timestamp}, bson.M{"$set": update}).Decode(&entry)
	//if err != nil {
	//	fmt.Println("Error finding and updating the entry in the database:", err)
	//	return types.UpdateEntryResponse{
	//		Success: false,
	//	}, err
	//}
	//
	//return types.UpdateEntryResponse{
	//	Success: true,
	//}, nil
}
