package main

import (
	"JourneyAppServer/aws"
	"JourneyAppServer/db"
	entriesHandlers "JourneyAppServer/handlers/entries"
	userHandlers "JourneyAppServer/handlers/users"
	"JourneyAppServer/middleware"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	db.MongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Successfully connected to MongoDB")

	defer func() {
		if err := db.MongoClient.Disconnect(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	// Login & Users
	http.HandleFunc("/api/validate/username", userHandlers.ValidateUsernameHandler)
	http.HandleFunc("/api/users/create", userHandlers.CreateUserHandler)
	http.HandleFunc("/api/users/login", userHandlers.LoginHandler)
	http.HandleFunc("/api/users/list", userHandlers.ListUsersHandler)
	http.HandleFunc("/api/users/get", userHandlers.GetUserHandler)
	// http.HandleFunc("/api/users/get", middleware.CombinedAuthMiddleware(userHandlers.GetUserHandler))
	//http.HandleFunc("/api/users/update", userHandlers.UpdateUserHandler)
	http.HandleFunc("/api/users/delete", middleware.CombinedAuthMiddleware(userHandlers.DeleteAccountHandler))

	// Entries
	http.HandleFunc("/api/entries/list", entriesHandlers.ListEntriesHandler) // no middleware here, it's being deprecated
	http.HandleFunc("/api/entries/create", middleware.CombinedAuthMiddleware(entriesHandlers.CreateNewEntryHandler))
	http.HandleFunc("/api/entries/update", middleware.CombinedAuthMiddleware(entriesHandlers.UpdateEntryHandler))
	http.HandleFunc("/api/entries/getPresignedPutURL", middleware.CombinedAuthMiddleware(aws.PresignPutHandler))
	http.HandleFunc("/api/entries/getPresignedGetURL", middleware.CombinedAuthMiddleware(aws.PresignGetHandler))
	http.HandleFunc("/api/entries/delete", entriesHandlers.DeleteEntryHandler)
	http.HandleFunc("/api/entries/search", middleware.CombinedAuthMiddleware(entriesHandlers.SearchEntriesHandler))
	http.HandleFunc("/api/entries/listUniqueLocations", middleware.CombinedAuthMiddleware(entriesHandlers.ListUniqueLocationsHandler))
	http.HandleFunc("/api/entries/listUniqueTags", middleware.CombinedAuthMiddleware(entriesHandlers.ListUniqueTagsHandler))
	//http.HandleFunc("/fix", entriesHandlers.FixTimestampHandler)

	fmt.Println("Server running on port 6913...")

	if err := http.ListenAndServe(":6913", nil); err != nil {
		log.Fatalf("Failed to start server on port 6913: %v", err)
	}
}
