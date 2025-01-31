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
	"time"
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

	response, err := listUniqueLocations(user)
	if err != nil {
		http.Error(w, "Error listing unique locations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func listUniqueLocations(user string) ([]types.LocationData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"username": user}}},
		{{Key: "$unwind", Value: "$locations"}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"latitude":    "$locations.latitude",
				"longitude":   "$locations.longitude",
				"displayName": "$locations.displayName",
			},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":         0,
			"latitude":    "$_id.latitude",
			"longitude":   "$_id.longitude",
			"displayName": "$_id.displayName",
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		fmt.Println("Error aggregating unique locations from database: ", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []types.LocationData
	if err := cursor.All(ctx, &results); err != nil {
		fmt.Println("Error decoding aggregated unique locations from cursor: ", err)
		return nil, err
	}

	return results, nil
}
