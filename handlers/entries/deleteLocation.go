package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"net/http"
	"strconv"
)

func DeleteLocationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.DeleteLocationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := deleteLocation(req, r)
	if err != nil {
		http.Error(w, "Error deleting the location", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteLocation(req types.DeleteLocationRequest, r *http.Request) (types.DeleteLocationResponse, error) {
	tx, err := db.SDB.Begin()
	if err != nil {
		utils.LM.Logger.Printf("Error starting transaction for location deletion: entry=%s, error=%v", req.EntryID, err)
		return types.DeleteLocationResponse{Success: false}, err
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
		return types.DeleteLocationResponse{Success: false}, err
	}
	if !exists {
		utils.LM.Logger.Printf("Entry not found for location deletion: entry=%s, userId=%s, timestamp=%v", req.EntryID, req.UserID, req.Timestamp)
		return types.DeleteLocationResponse{Success: false}, nil
	}

	deleteQuery := `
        DELETE FROM entry_locations 
        WHERE entry_id = ?
    `
	_, err = tx.Exec(deleteQuery, req.EntryID)
	if err != nil {
		utils.LM.Logger.Printf("Error deleting existing locations for entry %s: %v", req.EntryID, err)
		return types.DeleteLocationResponse{Success: false}, err
	}

	if len(req.Locations) > 0 {
		insertQuery := `
            INSERT INTO entry_locations (entry_id, latitude, longitude, display_name)
            VALUES (?, ?, ?, ?)
        `
		for _, loc := range req.Locations {
			_, err = tx.Exec(insertQuery, req.EntryID, loc.Latitude, loc.Longitude, loc.DisplayName)
			if err != nil {
				utils.LM.Logger.Printf("Error inserting new location for entry %s: lat=%f, lon=%f, name=%s, error=%v",
					req.EntryID, loc.Latitude, loc.Longitude, loc.DisplayName, err)
				return types.DeleteLocationResponse{Success: false}, err
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
		return types.DeleteLocationResponse{Success: false}, err
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
			"location_count": strconv.Itoa(len(req.Locations)),
		}
		metadataJSON, _ := json.Marshal(metadata)
		analyticsQuery := `
            INSERT INTO analytics_events (
                user_id, event_type, object_type, object_id, event_time, metadata
            ) VALUES (?, ?, ?, ?, NOW(), ?)
        `
		_, err := db.SDB.Exec(analyticsQuery, req.UserID, "delete_location", "entry", req.EntryID, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error for location deletion from entry %s: %v", req.EntryID, err)
		}
	}()

	utils.LM.Logger.Printf("Successfully updated locations for entry %s (new count: %d) for user %s", req.EntryID, len(req.Locations), req.UserID)
	return types.DeleteLocationResponse{Success: true}, nil

	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//
	//var entry types.Entry
	//err := collection.FindOneAndUpdate(ctx, bson.M{"id": req.EntryID, "userId": req.UserID, "timestamp": req.Timestamp}, bson.M{"$set": bson.M{"locations": req.Locations}}).Decode(&entry)
	//if err != nil {
	//	fmt.Println("Error deleting the tag from the entry in the database:", err)
	//	return types.DeleteLocationResponse{
	//		Success: false,
	//	}, err
	//}
	//fmt.Println("Delete location result is:", entry)
	//
	//return types.DeleteLocationResponse{
	//	Success: true,
	//}, nil
}
