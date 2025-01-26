package entriesHandlers

import (
	"JourneyAppServer/db"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"time"
)

func DeleteEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing required param \"id\".", http.StatusBadRequest)
		return
	}

	success, err := deleteEntry(id)
	if err != nil {
		http.Error(w, "Error deleting the entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": %v}`, success)
}

func deleteEntry(id string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	result, err := collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		fmt.Println("Error deleting the entry from the database:", err)
		return false, err
	}
	fmt.Println("Delete result is:", result)

	return true, nil
}
