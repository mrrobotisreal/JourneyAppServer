package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func SearchEntriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	pageStr := q.Get("page")
	limitStr := q.Get("limit")
	userStr := q.Get("user")

	var page int64
	var limit int64
	page = 1   // default of page 1
	limit = 20 // default of limit 20
	if pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 64); err == nil {
			page = p
		} else {
			utils.LM.Logger.Printf("Invalid page parameter: %s, error=%v", pageStr, err)
		}
	}
	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			limit = l
		} else {
			utils.LM.Logger.Printf("Invalid limit parameter: %s, error=%v", limitStr, err)
		}
	}

	var req types.SearchEntriesRequest
	req.Page = page
	req.Limit = limit
	req.User = userStr
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.LM.Logger.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.User == "" {
		http.Error(w, "Missing required query param \"user\"", http.StatusBadRequest)
		return
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 50 {
		req.Limit = 20
	}
	if req.Timeframe == "" {
		req.Timeframe = "All"
	}
	if req.SortRule == "" {
		req.SortRule = "Newest"
	}
	if req.Timeframe == "custom" && (req.FromDate == "" && req.ToDate == "") {
		http.Error(w, "Missing 'fromDate' or 'toDate' for custom timeframe", http.StatusBadRequest)
		return
	}

	response, err := searchEntries(req, r)
	if err != nil {
		utils.LM.Logger.Printf("Error searching entries for user %s: %v", req.User, err)
		http.Error(w, "Error aggregating search results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func searchEntries(req types.SearchEntriesRequest, r *http.Request) ([]types.Entry, error) {
	var query = `
        SELECT DISTINCT e.entry_id, e.user_id, e.username, e.text, e.timestamp, e.last_updated
        FROM entries e
    `
	var args []interface{}
	whereClauses := []string{"e.username = ?"}
	args = append(args, req.User)

	if req.SearchQuery != "" {
		whereClauses = append(whereClauses, "MATCH(e.text) AGAINST (? IN BOOLEAN MODE)")
		args = append(args, req.SearchQuery)
	}

	if len(req.Locations) > 0 {
		query += " LEFT JOIN entry_locations el ON e.entry_id = el.entry_id"
		var locConditions []string
		for _, loc := range req.Locations {
			locConditions = append(locConditions, "el.display_name = ?")
			args = append(args, loc.DisplayName)
		}
		whereClauses = append(whereClauses, "("+strings.Join(locConditions, " OR ")+")")
	}

	if len(req.Tags) > 0 {
		query += " LEFT JOIN entry_tags et ON e.entry_id = et.entry_id"
		var tagConditions []string
		for _, tag := range req.Tags {
			tagConditions = append(tagConditions, "et.tag_key = ?")
			args = append(args, tag.Key)
		}
		whereClauses = append(whereClauses, "("+strings.Join(tagConditions, " OR ")+")")
	}

	switch req.Timeframe {
	case "Past year":
		whereClauses = append(whereClauses, "e.timestamp >= ?")
		args = append(args, time.Now().AddDate(-1, 0, 0))
	case "Past 6 months":
		whereClauses = append(whereClauses, "e.timestamp >= ?")
		args = append(args, time.Now().AddDate(0, -6, 0))
	case "Past 3 months":
		whereClauses = append(whereClauses, "e.timestamp >= ?")
		args = append(args, time.Now().AddDate(0, -3, 0))
	case "Past 30 days":
		whereClauses = append(whereClauses, "e.timestamp >= ?")
		args = append(args, time.Now().AddDate(0, 0, -30))
	case "custom":
		if req.FromDate != "" {
			fromT, err := time.Parse(time.RFC3339, req.FromDate)
			if err == nil {
				whereClauses = append(whereClauses, "e.timestamp >= ?")
				args = append(args, fromT)
			}
		}
		if req.ToDate != "" {
			toT, err := time.Parse(time.RFC3339, req.ToDate)
			if err == nil {
				whereClauses = append(whereClauses, "e.timestamp <= ?")
				args = append(args, toT)
			}
		}
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	//if len(whereClauses) > 1 || (req.SearchQuery == "" && len(whereClauses) > 0) {
	//	query += " WHERE " + strings.Join(whereClauses, " AND ")
	//}

	sortDir := "DESC"
	if req.SortRule == "Oldest" {
		sortDir = "ASC"
	}
	query += " ORDER BY e.timestamp " + sortDir

	offset := (req.Page - 1) * req.Limit
	query += " LIMIT ? OFFSET ?"
	args = append(args, req.Limit, offset)

	rows, err := db.SDB.Query(query, args...)
	if err != nil {
		utils.LM.Logger.Printf("Error querying entries: user=%s, error=%v", req.User, err)
		return nil, err
	}
	defer rows.Close()

	var entries []types.Entry
	entryMap := make(map[string]*types.Entry)
	for rows.Next() {
		var e types.Entry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.Text, &e.Timestamp, &e.LastUpdated); err != nil {
			utils.LM.Logger.Printf("Error scanning entry row: user=%s, error=%v", req.User, err)
			return nil, err
		}
		entryMap[e.ID] = &e
	}
	if err := rows.Err(); err != nil {
		utils.LM.Logger.Printf("Row iteration error: user=%s, error=%v", req.User, err)
		return nil, err
	}

	for _, e := range entryMap {
		locQuery := "SELECT latitude, longitude, display_name FROM entry_locations WHERE entry_id = ?"
		locRows, err := db.SDB.Query(locQuery, e.ID)
		if err != nil {
			utils.LM.Logger.Printf("Error querying locations for entry %s: %v", e.ID, err)
			return nil, err
		}
		defer locRows.Close()
		for locRows.Next() {
			var loc types.LocationData
			if err := locRows.Scan(&loc.Latitude, &loc.Longitude, &loc.DisplayName); err != nil {
				utils.LM.Logger.Printf("Error scanning location for entry %s: %v", e.ID, err)
				return nil, err
			}
			e.Locations = append(e.Locations, loc)
		}

		tagQuery := "SELECT tag_key, tag_value FROM entry_tags WHERE entry_id = ?"
		tagRows, err := db.SDB.Query(tagQuery, e.ID)
		if err != nil {
			utils.LM.Logger.Printf("Error querying tags for entry %s: %v", e.ID, err)
			return nil, err
		}
		defer tagRows.Close()
		for tagRows.Next() {
			var tag types.TagData
			if err := tagRows.Scan(&tag.Key, &tag.Value); err != nil {
				utils.LM.Logger.Printf("Error scanning tag for entry %s: %v", e.ID, err)
				return nil, err
			}
			e.Tags = append(e.Tags, tag)
		}

		imgQuery := "SELECT image_url FROM entry_images WHERE entry_id = ?"
		imgRows, err := db.SDB.Query(imgQuery, e.ID)
		if err != nil {
			utils.LM.Logger.Printf("Error querying images for entry %s: %v", e.ID, err)
			return nil, err
		}
		defer imgRows.Close()
		for imgRows.Next() {
			var img string
			if err := imgRows.Scan(&img); err != nil {
				utils.LM.Logger.Printf("Error scanning image for entry %s: %v", e.ID, err)
				return nil, err
			}
			e.Images = append(e.Images, img)
		}

		entries = append(entries, *e)
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
			"search_query": req.SearchQuery,
			"timeframe":    req.Timeframe,
			"sort_rule":    req.SortRule,
			"page":         strconv.FormatInt(req.Page, 10),
			"limit":        strconv.FormatInt(req.Limit, 10),
		}
		metadataJSON, _ := json.Marshal(metadata)
		analyticsQuery := `
            INSERT INTO analytics_events (
                user_id, event_type, object_type, object_id, event_time, metadata
            ) VALUES (?, ?, ?, ?, NOW(), ?)
        `
		_, err := db.SDB.Exec(analyticsQuery, req.User, "search_entries", "entries", "all", string(metadataJSON))
		if err != nil {
			utils.LM.Logger.Printf("Analytics logging error for search entries user %s: %v", req.User, err)
		}
	}()

	utils.LM.Logger.Printf("Successfully searched entries for user %s: page=%d, limit=%d, count=%d", req.User, req.Page, req.Limit, len(entries))
	return entries, nil

	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	//pipeline := mongo.Pipeline{}
	//
	//matchUser := bson.D{{Key: "$match", Value: bson.D{
	//	{Key: "username", Value: req.User},
	//}}}
	//pipeline = append(pipeline, matchUser)
	//
	//if req.SearchQuery != "" {
	//	matchSearch := bson.D{{"$match", bson.D{
	//		{"text", bson.D{
	//			{"$regex", primitive.Regex{Pattern: req.SearchQuery, Options: "i"}},
	//		}},
	//	}}}
	//	pipeline = append(pipeline, matchSearch)
	//}
	//
	//if len(req.Locations) > 0 {
	//	var orArray []bson.M
	//	for _, loc := range req.Locations {
	//		orArray = append(orArray, bson.M{
	//			"locations": bson.M{
	//				"$elemMatch": bson.M{"displayName": loc.DisplayName},
	//			},
	//		})
	//	}
	//	matchLoc := bson.D{{"$match", bson.D{
	//		{"$or", orArray},
	//	}}}
	//	pipeline = append(pipeline, matchLoc)
	//}
	//
	//if len(req.Tags) > 0 {
	//	var orArray []bson.M
	//	for _, tag := range req.Tags {
	//		orArray = append(orArray, bson.M{
	//			"tags": bson.M{
	//				"$elemMatch": bson.M{"key": tag.Key},
	//			},
	//		})
	//	}
	//	matchTags := bson.D{{"$match", bson.D{
	//		{"$or", orArray},
	//	}}}
	//	pipeline = append(pipeline, matchTags)
	//}
	//
	//switch req.Timeframe {
	//case "Past year":
	//	cutoff := time.Now().AddDate(-1, 0, 0)
	//	pipeline = append(pipeline,
	//		bson.D{{"$match", bson.D{
	//			{"timestamp", bson.D{{"$gte", cutoff}}},
	//		}}})
	//case "Past 6 months":
	//	cutoff := time.Now().AddDate(0, -6, 0)
	//	pipeline = append(pipeline,
	//		bson.D{{"$match", bson.D{
	//			{"timestamp", bson.D{{"$gte", cutoff}}},
	//		}}})
	//case "Past 3 months":
	//	cutoff := time.Now().AddDate(0, -3, 0)
	//	pipeline = append(pipeline,
	//		bson.D{{"$match", bson.D{
	//			{"timestamp", bson.D{{"$gte", cutoff}}},
	//		}}})
	//case "Past 30 days":
	//	cutoff := time.Now().AddDate(0, 0, -30)
	//	pipeline = append(pipeline,
	//		bson.D{{"$match", bson.D{
	//			{"timestamp", bson.D{{"$gte", cutoff}}},
	//		}}})
	//case "custom":
	//	fromT, _ := time.Parse(time.RFC3339, req.FromDate)
	//	toT, _ := time.Parse(time.RFC3339, req.ToDate)
	//	timeFilter := bson.M{}
	//	if !fromT.IsZero() {
	//		timeFilter["$gte"] = fromT
	//	}
	//	if !toT.IsZero() {
	//		timeFilter["$lte"] = toT
	//	}
	//	if len(timeFilter) > 0 {
	//		pipeline = append(pipeline, bson.D{{"$match", bson.D{
	//			{"timestamp", timeFilter},
	//		}}})
	//	}
	//case "All":
	//	// do nothing...
	//}
	//
	//sortDir := -1
	//if req.SortRule == "Oldest" {
	//	sortDir = 1
	//}
	//pipeline = append(pipeline, bson.D{{"$sort", bson.D{
	//	{"timestamp", sortDir},
	//}}})
	//
	//skip := (req.Page - 1) * req.Limit
	//pipeline = append(pipeline, bson.D{{"$skip", skip}})
	//pipeline = append(pipeline, bson.D{{"$limit", req.Limit}})
	//
	//cursor, err := collection.Aggregate(ctx, pipeline)
	//if err != nil {
	//	fmt.Println("Aggregate error: ", err)
	//	return nil, err
	//}
	//defer cursor.Close(ctx)
	//
	//var results []bson.M
	//if err := cursor.All(ctx, &results); err != nil {
	//	fmt.Println("Error reading cursor results: ", err)
	//	return nil, err
	//}
	//
	//return results, nil
}
