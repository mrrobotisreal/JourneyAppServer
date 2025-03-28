package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
)

func CreateNewEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.CreateNewEntryRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Locations == nil || len(req.Locations) <= 0 {
		req.Locations = make([]types.LocationData, 0)
	}
	if req.Tags == nil || len(req.Tags) <= 0 {
		req.Tags = make([]types.TagData, 0)
	}
	if req.Images == nil || len(req.Images) <= 0 {
		req.Images = make([]string, 0)
	}

	response, err := createNewEntry(req, r)
	if err != nil {
		http.Error(w, "Error creating new entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createNewEntry(req types.CreateNewEntryRequest, r *http.Request) (types.CreateNewEntryResponse, error) {
	entryID := uuid.New().String()

	tx, err := db.SDB.Begin()
	if err != nil {
		return types.CreateNewEntryResponse{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	entryQuery := `
        INSERT INTO entries (entry_id, user_id, username, text, timestamp)
        VALUES (?, ?, ?, ?, ?)
    `
	_, err = tx.Exec(entryQuery, entryID, req.UserID, req.Username, req.Text, req.Timestamp)
	if err != nil {
		utils.LM.Logger.Printf("Error inserting entry into database: user=%s, error=%v", req.Username, err)
		return types.CreateNewEntryResponse{}, err
	}

	for _, loc := range req.Locations {
		locQuery := `
            INSERT INTO entry_locations (entry_id, latitude, longitude, display_name)
            VALUES (?, ?, ?, ?)
        `
		_, err = tx.Exec(locQuery, entryID, loc.Latitude, loc.Longitude, loc.DisplayName)
		if err != nil {
			utils.LM.Logger.Printf("Error inserting location for entry %s: %v", entryID, err)
			return types.CreateNewEntryResponse{}, err
		}
	}

	for _, tag := range req.Tags {
		tagQuery := `
            INSERT INTO entry_tags (entry_id, tag_key, tag_value)
            VALUES (?, ?, ?)
        `
		_, err = tx.Exec(tagQuery, entryID, tag.Key, tag.Value)
		if err != nil {
			utils.LM.Logger.Printf("Error inserting tag for entry %s: %v", entryID, err)
			return types.CreateNewEntryResponse{}, err
		}
	}

	for _, image := range req.Images {
		imageQuery := `
            INSERT INTO entry_images (entry_id, image_url)
            VALUES (?, ?)
        `
		_, err = tx.Exec(imageQuery, entryID, image)
		if err != nil {
			utils.LM.Logger.Printf("Error inserting image for entry %s: %v", entryID, err)
			return types.CreateNewEntryResponse{}, err
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
		_, err := db.SDB.Exec(analyticsQuery, req.UserID, "create entry", "entry", entryID, string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error for entry %s: %v", entryID, err)
		}
	}()

	utils.LM.Logger.Printf("Successfully created entry %s for user %s", entryID, req.Username)
	return types.CreateNewEntryResponse{
		ID:          entryID,
		UserID:      req.UserID,
		Username:    req.Username,
		Text:        req.Text,
		Timestamp:   req.Timestamp,
		LastUpdated: req.Timestamp,
		Locations:   req.Locations,
		Tags:        req.Tags,
	}, nil

	//newEntry := types.Entry{
	//	ID:        uuid.New().String(),
	//	UserID:    req.UserID,
	//	Username:  req.Username,
	//	Text:      req.Text,
	//	Timestamp: req.Timestamp,
	//	Locations: req.Locations,
	//	Tags:      req.Tags,
	//}
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//
	//result, err := collection.InsertOne(ctx, newEntry)
	//if err != nil {
	//	fmt.Println("Error inserting new entry into the database:", err)
	//	return types.CreateNewEntryResponse{}, err
	//}
	//fmt.Println("Result:", result)
	//
	//return types.CreateNewEntryResponse{
	//	ID: newEntry.ID,
	//	UserID:    req.UserID,
	//	Username:  req.Username,
	//	Text:      req.Text,
	//	Timestamp: req.Timestamp,
	//	LastUpdated: req.Timestamp,
	//	Locations: req.Locations,
	//	Tags:      req.Tags,
	//}, nil
}
