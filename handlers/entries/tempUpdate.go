package entriesHandlers

import (
	"JourneyAppServer/db"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"
)

func FixTimestampHandler(w http.ResponseWriter, r *http.Request) {
	response, err := fixTimestamp()
	if err != nil {
		http.Error(w, "Error fixing timestamp", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": %v`, response)
}

func fixTimestamp() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	filter := bson.D{}

	updatePipeline := mongo.Pipeline{
		{
			{"$set", bson.D{
				{"$toDate", "$timestamp"},
			}},
		},
	}

	res, err := collection.UpdateMany(ctx, filter, updatePipeline)
	if err != nil {
		fmt.Println("Error updating all timestamps! ", err)
		return false, nil
	}

	fmt.Printf("Modified count: %d\n", res.ModifiedCount)
	return true, nil
}
