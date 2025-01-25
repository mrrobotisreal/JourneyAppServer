package userHandlers

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

func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	response, err := listUsers()
	if err != nil {
		http.Error(w, "Error listing users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func listUsers() ([]types.UserListItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)
	pipeline := mongo.Pipeline{
		{{"$sort", bson.D{{"username", 1}}}},
		{{"$project", bson.M{
			"username": 1,
			"_id":      0,
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		fmt.Println("Error aggregating users:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []types.UserListItem
	if err := cursor.All(ctx, &results); err != nil {
		fmt.Println("Error getting all users from the database:", err)
		return nil, err
	}

	return results, nil
}
