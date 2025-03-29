package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"net/http"
)

func ListUniqueTagsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	user := r.URL.Query().Get("user")
	if user == "" {
		http.Error(w, "Missing required query param \"user\"", http.StatusBadRequest)
		return
	}

	response, err := listUniqueTags(user, r)
	if err != nil {
		http.Error(w, "Error listing unique tags", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func listUniqueTags(user string, r *http.Request) ([]types.TagData, error) {
	query := `
        SELECT DISTINCT et.tag_key, et.tag_value
        FROM entry_tags et
        JOIN entries e ON et.entry_id = e.entry_id
        WHERE e.username = ?
        ORDER BY et.tag_key
    `
	rows, err := db.SDB.Query(query, user)
	if err != nil {
		utils.LM.Logger.Printf("Error querying unique tags for user %s: %v", user, err)
		return nil, err
	}
	defer rows.Close()

	var tags []types.TagData
	for rows.Next() {
		var tag types.TagData
		if err := rows.Scan(&tag.Key, &tag.Value); err != nil {
			utils.LM.Logger.Printf("Error scanning tag for user %s: %v", user, err)
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		utils.LM.Logger.Printf("Row iteration error for unique tags, user %s: %v", user, err)
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
	//	_, err := db.SDB.Exec(analyticsQuery, user, "list_unique_tags", "tags", "all", string(metadataJSON))
	//	if err != nil {
	//		utils.LM.Logger.Printf("Analytics logging error for list unique tags user %s: %v", user, err)
	//	}
	//}()

	utils.LM.Logger.Printf("Successfully retrieved %d unique tags for user %s", len(tags), user)
	return tags, nil

	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//pipeline := mongo.Pipeline{
	//	{{Key: "$match", Value: bson.M{"username": user}}},
	//	{{Key: "$unwind", Value: "$tags"}},
	//	{{Key: "$group", Value: bson.M{
	//		"_id":   bson.M{"key": "$tags.key"},
	//		"value": bson.M{"$first": "$tags.value"},
	//	}}},
	//	{{Key: "$project", Value: bson.M{
	//		"_id":   0,
	//		"key":   "$_id.key",
	//		"value": "$value",
	//	}}},
	//}
	//
	//cursor, err := collection.Aggregate(ctx, pipeline)
	//if err != nil {
	//	fmt.Println("Error aggregating tags from the database: ", err)
	//	return nil, err
	//}
	//defer cursor.Close(ctx)
	//
	//var results []types.TagData
	//if err := cursor.All(ctx, &results); err != nil {
	//	fmt.Println("Error reading results from cursor: ", err)
	//	return nil, err
	//}
	//
	//return results, nil
}
