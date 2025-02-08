package entriesHandlers

import (
	"JourneyAppServer/db"
	"context"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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
	user := r.URL.Query().Get("user")
	if user == "" {
		http.Error(w, "Missing required param \"user\".", http.StatusBadRequest)
		return
	}

	success, err := deleteEntry(id, user)
	if err != nil {
		http.Error(w, "Error deleting the entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": %v}`, success)
}

func deleteEntry(id, user string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoClient.Database(db.DbName).Collection(db.EntriesCollection)

	result, err := collection.DeleteOne(ctx, bson.M{"id": id, "username": user})
	if err != nil {
		fmt.Println("Error deleting the entry from the database:", err)
		return false, err
	}
	fmt.Println("Delete result is:", result)

	return true, nil
}
