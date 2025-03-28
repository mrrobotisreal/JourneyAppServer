package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func GetEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	userId := r.URL.Query().Get("userId")
	timestampStr := r.URL.Query().Get("timestamp")
	if id == "" {
		http.Error(w, "Missing required param \"id\"", http.StatusBadRequest)
		return
	}
	if userId == "" {
		http.Error(w, "Missing required param \"userId\"", http.StatusBadRequest)
		return
	}
	if timestampStr == "" {
		http.Error(w, "Missing required param \"timestamp\"", http.StatusBadRequest)
		return
	}
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		http.Error(w, "Invalid timestamp format", http.StatusBadRequest)
		return
	}

	response, err := getEntry(id, userId, timestamp, r)
	if err != nil {
		http.Error(w, "Error getting the entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getEntry(id, userId string, timestamp time.Time, r *http.Request) (types.Entry, error) {
	var entry types.Entry

	// Fetch the base entry data
	query := `
        SELECT entry_id, user_id, username, text, timestamp, last_updated
        FROM entries 
        WHERE entry_id = ? AND user_id = ? AND timestamp = ?
    `
	err := db.SDB.QueryRow(query, id, userId, timestamp).Scan(
		&entry.ID, &entry.UserID, &entry.Username, &entry.Text, &entry.Timestamp, &entry.LastUpdated,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.LM.Logger.Printf("Entry not found: id=%s, userId=%s, timestamp=%v", id, userId, timestamp)
			return types.Entry{}, fmt.Errorf("entry not found")
		}
		utils.LM.Logger.Printf("Error querying entry from database: id=%s, userId=%s, timestamp=%v, error=%v", id, userId, timestamp, err)
		return types.Entry{}, err
	}

	// Fetch locations
	locQuery := `
        SELECT latitude, longitude, display_name 
        FROM entry_locations 
        WHERE entry_id = ?
    `
	locRows, err := db.SDB.Query(locQuery, id)
	if err != nil {
		utils.LM.Logger.Printf("Error querying locations for entry %s: %v", id, err)
		return types.Entry{}, err
	}
	defer locRows.Close()

	entry.Locations = []types.LocationData{}
	for locRows.Next() {
		var loc types.LocationData
		if err := locRows.Scan(&loc.Latitude, &loc.Longitude, &loc.DisplayName); err != nil {
			utils.LM.Logger.Printf("Error scanning location for entry %s: %v", id, err)
			return types.Entry{}, err
		}
		entry.Locations = append(entry.Locations, loc)
	}
	if err := locRows.Err(); err != nil {
		utils.LM.Logger.Printf("Location row iteration error for entry %s: %v", id, err)
		return types.Entry{}, err
	}

	// Fetch tags
	tagQuery := `
        SELECT tag_key, tag_value 
        FROM entry_tags 
        WHERE entry_id = ?
    `
	tagRows, err := db.SDB.Query(tagQuery, id)
	if err != nil {
		utils.LM.Logger.Printf("Error querying tags for entry %s: %v", id, err)
		return types.Entry{}, err
	}
	defer tagRows.Close()

	entry.Tags = []types.TagData{}
	for tagRows.Next() {
		var tag types.TagData
		if err := tagRows.Scan(&tag.Key, &tag.Value); err != nil {
			utils.LM.Logger.Printf("Error scanning tag for entry %s: %v", id, err)
			return types.Entry{}, err
		}
		entry.Tags = append(entry.Tags, tag)
	}
	if err := tagRows.Err(); err != nil {
		utils.LM.Logger.Printf("Tag row iteration error for entry %s: %v", id, err)
		return types.Entry{}, err
	}

	// Fetch images
	imgQuery := `
        SELECT image_url 
        FROM entry_images 
        WHERE entry_id = ?
    `
	imgRows, err := db.SDB.Query(imgQuery, id)
	if err != nil {
		utils.LM.Logger.Printf("Error querying images for entry %s: %v", id, err)
		return types.Entry{}, err
	}
	defer imgRows.Close()

	entry.Images = []string{}
	for imgRows.Next() {
		var image string
		if err := imgRows.Scan(&image); err != nil {
			utils.LM.Logger.Printf("Error scanning image for entry %s: %v", id, err)
			return types.Entry{}, err
		}
		entry.Images = append(entry.Images, image)
	}
	if err := imgRows.Err(); err != nil {
		utils.LM.Logger.Printf("Image row iteration error for entry %s: %v", id, err)
		return types.Entry{}, err
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
		_, err := db.SDB.Exec(analyticsQuery, userId, "view entry", "entry", id, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error for entry %s: %v", id, err)
		}
	}()

	utils.LM.Logger.Printf("Successfully retrieved entry: id=%s, userId=%s, timestamp=%v", id, userId, timestamp)
	return entry, nil

	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//
	//var entry types.Entry
	//err := collection.FindOne(ctx, bson.M{"id": id, "userId": userId, "timestamp": timestamp}).Decode(&entry)
	//if err != nil {
	//	fmt.Println("Error finding the entry in the database: ", err)
	//	return types.Entry{}, err
	//}
	//
	//return entry, nil
}
