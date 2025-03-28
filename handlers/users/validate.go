package userHandlers

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

func ValidateUsernameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req types.ValidateUsernameRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Println("Username: ", req.Username)

	response, err := validateUsername(req)
	if err != nil {
		http.Error(w, "Error validating username", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func validateUsername(req types.ValidateUsernameRequest) (types.ValidateUsernameResponse, error) {
	var exists bool

	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)`

	err := db.SDB.QueryRow(query, req.Username).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.ValidateUsernameResponse{
				UsernameAvailable: true,
			}, nil
		}
		return types.ValidateUsernameResponse{
			UsernameAvailable: false,
		}, err
	}

	return types.ValidateUsernameResponse{
		UsernameAvailable: !exists,
	}, nil

	//------------------------Below is old MongoDB code-------------------------------------------//

	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//
	//collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)
	//
	//var userResult types.User
	//err = collection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&userResult)
	//if err != nil {
	//	if err == mongo.ErrNoDocuments {
	//		return types.ValidateUsernameResponse{
	//			UsernameAvailable: true,
	//		}, nil
	//	}
	//
	//	fmt.Println("Error finding user in the database:", err)
	//	return types.ValidateUsernameResponse{
	//		UsernameAvailable: false,
	//	}, err
	//}
	//
	//return types.ValidateUsernameResponse{
	//	UsernameAvailable: false,
	//}, nil
}
