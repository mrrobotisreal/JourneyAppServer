package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strconv"
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
			// TODO: handle error
		}
	}
	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			limit = l
		} else {
			// TODO: handle error
		}
	}

	var req types.SearchEntriesRequest
	req.Page = page
	req.Limit = limit
	req.User = userStr
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	response, err := searchEntries(req)
	if err != nil {
		http.Error(w, "Error aggregating search results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func searchEntries(req types.SearchEntriesRequest) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	pipeline := mongo.Pipeline{}

	matchUser := bson.D{{Key: "$match", Value: bson.D{
		{Key: "username", Value: req.User},
	}}}
	pipeline = append(pipeline, matchUser)

	if req.SearchQuery != "" {
		matchSearch := bson.D{{"$match", bson.D{
			{"text", bson.D{
				{"$regex", primitive.Regex{Pattern: req.SearchQuery, Options: "i"}},
			}},
		}}}
		pipeline = append(pipeline, matchSearch)
	}

	if len(req.Locations) > 0 {
		var orArray []bson.M
		for _, loc := range req.Locations {
			orArray = append(orArray, bson.M{
				"locations": bson.M{
					"$elemMatch": bson.M{"displayName": loc.DisplayName},
				},
			})
		}
		matchLoc := bson.D{{"$match", bson.D{
			{"$or", orArray},
		}}}
		pipeline = append(pipeline, matchLoc)
	}

	if len(req.Tags) > 0 {
		var orArray []bson.M
		for _, tag := range req.Tags {
			orArray = append(orArray, bson.M{
				"tags": bson.M{
					"$elemMatch": bson.M{"key": tag.Key},
				},
			})
		}
		matchTags := bson.D{{"$match", bson.D{
			{"$or", orArray},
		}}}
		pipeline = append(pipeline, matchTags)
	}

	switch req.Timeframe {
	case "Past year":
		cutoff := time.Now().AddDate(-1, 0, 0)
		pipeline = append(pipeline,
			bson.D{{"$match", bson.D{
				{"timestamp", bson.D{{"$gte", cutoff}}},
			}}})
	case "Past 6 months":
		cutoff := time.Now().AddDate(0, -6, 0)
		pipeline = append(pipeline,
			bson.D{{"$match", bson.D{
				{"timestamp", bson.D{{"$gte", cutoff}}},
			}}})
	case "Past 3 months":
		cutoff := time.Now().AddDate(0, -3, 0)
		pipeline = append(pipeline,
			bson.D{{"$match", bson.D{
				{"timestamp", bson.D{{"$gte", cutoff}}},
			}}})
	case "Past 30 days":
		cutoff := time.Now().AddDate(0, 0, -30)
		pipeline = append(pipeline,
			bson.D{{"$match", bson.D{
				{"timestamp", bson.D{{"$gte", cutoff}}},
			}}})
	case "custom":
		fromT, _ := time.Parse(time.RFC3339, req.FromDate)
		toT, _ := time.Parse(time.RFC3339, req.ToDate)
		timeFilter := bson.M{}
		if !fromT.IsZero() {
			timeFilter["$gte"] = fromT
		}
		if !toT.IsZero() {
			timeFilter["$lte"] = toT
		}
		if len(timeFilter) > 0 {
			pipeline = append(pipeline, bson.D{{"$match", bson.D{
				{"timestamp", timeFilter},
			}}})
		}
	case "All":
		// do nothing...
	}

	sortDir := -1
	if req.SortRule == "Oldest" {
		sortDir = 1
	}
	pipeline = append(pipeline, bson.D{{"$sort", bson.D{
		{"timestamp", sortDir},
	}}})

	skip := (req.Page - 1) * req.Limit
	pipeline = append(pipeline, bson.D{{"$skip", skip}})
	pipeline = append(pipeline, bson.D{{"$limit", req.Limit}})

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		fmt.Println("Aggregate error: ", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		fmt.Println("Error reading cursor results: ", err)
		return nil, err
	}

	return results, nil
}
