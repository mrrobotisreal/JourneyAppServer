package entriesHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func ListEntriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	user := r.URL.Query().Get("user")
	sortRule := r.URL.Query().Get("sortRule")

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		http.Error(w, "Missing param \"limit\". \"limit\" is required.", http.StatusBadRequest)
		return
	}
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		http.Error(w, "Error converting \"limitStr\" to int", http.StatusInternalServerError)
		return
	}

	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		http.Error(w, "Missing param \"page\". \"page\" is required.", http.StatusBadRequest)
		return
	}
	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil {
		http.Error(w, "Error converting \"pageStr\" to int", http.StatusInternalServerError)
		return
	}

	response, err := listEntries(types.ListEntriesParams{
		User:      user,
		Locations: []types.LocationData{}, // TODO: implement this later, possibly switch to request over params
		Tags:      []types.TagData{},      // TODO: implement this later, possibly switch to request over params
		Limit:     limit,
		Page:      page,
		SortRule:  sortRule,
	})
	if err != nil {
		http.Error(w, "Error listing entries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func listEntries(params types.ListEntriesParams) ([]types.EntryListItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	pipeline := mongo.Pipeline{}
	matchStage := bson.D{
		{"$match", bson.D{
			{"username", params.User},
		}},
	}

	if len(params.Locations) > 0 {
		locationOrArray := make([]bson.M, 0, len(params.Locations))
		for _, loc := range params.Locations {
			locationOrArray = append(locationOrArray, bson.M{
				"locations": bson.M{
					"$elemMatch": bson.M{
						"latitude":  loc.Latitude,
						"longitude": loc.Longitude,
					},
				},
			})
		}
		// TODO: the line below is incorrect and needs to be fixed
		// matchStage[0].Value.(bson.D).Append("$or", locationOrArray)
	}

	if len(params.Tags) > 0 {
		tagAndArray := make([]bson.M, 0, len(params.Tags))
		for _, t := range params.Tags {
			if t.Value == "" {
				tagAndArray = append(tagAndArray, bson.M{
					"tags": bson.M{
						"$elemMatch": bson.M{"key": t.Key},
					},
				})
			} else {
				tagAndArray = append(tagAndArray, bson.M{
					"tags": bson.M{
						"$elemMatch": bson.M{
							"key":   t.Key,
							"value": t.Value,
						},
					},
				})
			}
		}
		// TODO: the line below is incorrect and needs to be fixed
		// matchStage[0].Value.(bson.D).Append("$and", tagAndArray)
	}

	pipeline = append(pipeline, matchStage)

	sortKey := -1
	if strings.ToLower(params.SortRule) == "oldest" {
		sortKey = 1
	}
	sortStage := bson.D{{"$sort", bson.D{{"timestamp", sortKey}}}}
	pipeline = append(pipeline, sortStage)

	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 50 {
		params.Limit = 50
	}
	if params.Page < 1 {
		params.Page = 1
	}

	skipValue := (params.Page - 1) * params.Limit

	skipStage := bson.D{{"$skip", skipValue}}
	limitStage := bson.D{{"$limit", params.Limit}}

	pipeline = append(pipeline, skipStage, limitStage)

	projectStage := bson.D{{
		"$project", bson.D{
			{"id", "$uuid"},
			{"text", 1},
			{"timestamp", 1},
			{"locations", 1},
			{"tags", 1},
			{"images", 1},
			{"_id", 0},
		},
	}}
	pipeline = append(pipeline, projectStage)

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate error: %w", err)
	}
	defer cursor.Close(ctx)

	var results []types.EntryListItem
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("cursor.All error: %w", err)
	}

	return results, nil
}
