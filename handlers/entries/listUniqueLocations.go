package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"net/http"
)

func ListUniqueLocationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	user := r.URL.Query().Get("user")
	if user == "" {
		http.Error(w, "Missing required query param \"user\"", http.StatusBadRequest)
		return
	}

	response, err := listUniqueLocations(user, r)
	if err != nil {
		http.Error(w, "Error listing unique locations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func listUniqueLocations(user string, r *http.Request) ([]types.LocationData, error) {
	query := `
        SELECT DISTINCT el.latitude, el.longitude, el.display_name
        FROM entry_locations el
        JOIN entries e ON el.entry_id = e.entry_id
        WHERE e.username = ?
        ORDER BY el.display_name
    `
	rows, err := db.SDB.Query(query, user)
	if err != nil {
		utils.LM.Logger.Printf("Error querying unique locations for user %s: %v", user, err)
		return nil, err
	}
	defer rows.Close()

	var locations []types.LocationData
	for rows.Next() {
		var loc types.LocationData
		if err := rows.Scan(&loc.Latitude, &loc.Longitude, &loc.DisplayName); err != nil {
			utils.LM.Logger.Printf("Error scanning location for user %s: %v", user, err)
			return nil, err
		}
		locations = append(locations, loc)
	}
	if err := rows.Err(); err != nil {
		utils.LM.Logger.Printf("Row iteration error for unique locations, user %s: %v", user, err)
		return nil, err
	}

	//go func() {
	//	ip := r.Header.Get("X-Forwarded-For")
	//	if ip == "" {
	//		ip = r.RemoteAddr
	//	}
	//	metadata := map[string]string{
	//		"source":       "api",
	//		"client_ip":    ip,
	//		"user_agent":   r.Header.Get("User-Agent"),
	//		"app_version":  r.Header.Get("X-App-Version"),
	//		"os_version":   r.Header.Get("X-OS-Version"),
	//		"device_model": r.Header.Get("X-Device-Model"),
	//	}
	//	metadataJSON, _ := json.Marshal(metadata)
	//	analyticsQuery := `
	//        INSERT INTO analytics_events (
	//            user_id, event_type, object_type, object_id, event_time, meta_data
	//        ) VALUES (?, ?, ?, ?, NOW(), ?)
	//    `
	//	_, err := db.SDB.Exec(analyticsQuery, user, "list_unique_locations", "locations", "all", string(metadataJSON))
	//	if err != nil {
	//		utils.LM.Logger.Printf("Analytics logging error for list unique locations user %s: %v", user, err)
	//	}
	//}()

	utils.LM.Logger.Printf("Successfully retrieved %d unique locations for user %s", len(locations), user)
	return locations, nil

	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//pipeline := mongo.Pipeline{
	//	{{Key: "$match", Value: bson.M{"username": user}}},
	//	{{Key: "$unwind", Value: "$locations"}},
	//	{{Key: "$group", Value: bson.M{
	//		"_id": bson.M{
	//			"latitude":    "$locations.latitude",
	//			"longitude":   "$locations.longitude",
	//			"displayName": "$locations.displayName",
	//		},
	//	}}},
	//	{{Key: "$project", Value: bson.M{
	//		"_id":         0,
	//		"latitude":    "$_id.latitude",
	//		"longitude":   "$_id.longitude",
	//		"displayName": "$_id.displayName",
	//	}}},
	//}
	//
	//cursor, err := collection.Aggregate(ctx, pipeline)
	//if err != nil {
	//	fmt.Println("Error aggregating unique locations from database: ", err)
	//	return nil, err
	//}
	//defer cursor.Close(ctx)
	//
	//var results []types.LocationData
	//if err := cursor.All(ctx, &results); err != nil {
	//	fmt.Println("Error decoding aggregated unique locations from cursor: ", err)
	//	return nil, err
	//}
	//
	//return results, nil
}
