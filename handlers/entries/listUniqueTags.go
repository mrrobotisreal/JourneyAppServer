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

	response, err := listUniqueTags(user)
	if err != nil {
		http.Error(w, "Error listing unique tags", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func listUniqueTags(user string) ([]types.TagData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"username": user}}},
		{{Key: "$unwind", Value: "$tags"}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"key": "$tags.key"},
			"value": bson.M{"$first": "$tags.value"},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":   0,
			"key":   "$_id.key",
			"value": "$value",
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		fmt.Println("Error aggregating tags from the database: ", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []types.TagData
	if err := cursor.All(ctx, &results); err != nil {
		fmt.Println("Error reading results from cursor: ", err)
		return nil, err
	}

	return results, nil
}
