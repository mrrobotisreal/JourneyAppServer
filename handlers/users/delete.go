package userHandlers

import (
	"JourneyAppServer/aws"
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"time"
)

func DeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("user")
	if username == "" {
		http.Error(w, "Missing required param \"user\"", http.StatusBadRequest)
		return
	}

	response, err := deleteAccount(username)
	if err != nil {
		http.Error(w, "Error deleting account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteAccount(username string) (types.DeleteAccountResponse, error) {
	// first go through and delete all journal entries associated with the account
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userCollection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)
	entryCollection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	pipeline := mongo.Pipeline{}

	matchUser := bson.D{{Key: "$match", Value: bson.D{
		{Key: "username", Value: username},
	}}}
	pipeline = append(pipeline, matchUser)

	cursor, err := entryCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Aggregate all entries error: ", err)
		return types.DeleteAccountResponse{
			Success: false,
		}, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var EntryImages struct {
			IDM    string   `bson:"_id"`
			ID     string   `bson:"id"`
			Images []string `bson:"images"`
		}
		if err := cursor.Decode(&EntryImages); err != nil {
			log.Println("Error decoding individual entry:", err)
			return types.DeleteAccountResponse{
				Success: false,
			}, err
		}

		if len(EntryImages.Images) > 0 {
			for _, imageKey := range EntryImages.Images {
				if res := aws.DeleteImage(imageKey); !res.Success {
					log.Println("Error deleting individual image: ", imageKey)
				} else {
					log.Println("Successfully deleted image: ", imageKey, "!")
				}
			}
		}
	}
	if err := cursor.Err(); err != nil {
		log.Println("Cursor encountered an error: ", err)
		return types.DeleteAccountResponse{
			Success: false,
		}, err
	}

	_, err = entryCollection.DeleteMany(ctx, bson.M{"username": username})
	if err != nil {
		fmt.Println("Error deleting all entries from the database: ", err)
		return types.DeleteAccountResponse{
			Success: false,
		}, err
	}

	// then delete the account
	_, err = userCollection.DeleteOne(ctx, bson.M{"username": username})
	if err != nil {
		fmt.Println("Error deleting the user from the database: ", err)
		return types.DeleteAccountResponse{
			Success: false,
		}, err
	}

	return types.DeleteAccountResponse{
		Success: true,
	}, nil
}
