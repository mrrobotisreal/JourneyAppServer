package main

import (
	"JourneyAppServer/aws"
	"JourneyAppServer/db"
	entriesHandlers "JourneyAppServer/handlers/entries"
	userHandlers "JourneyAppServer/handlers/users"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	db.MongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		log.Fatal(err)
	}

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

	// Entries
	http.HandleFunc("/api/entries/list", entriesHandlers.ListEntriesHandler)
	http.HandleFunc("/api/entries/create", entriesHandlers.CreateNewEntryHandler)
	http.HandleFunc("/api/entries/update", entriesHandlers.UpdateEntryHandler)
	http.HandleFunc("/api/entries/getPresignedURL", aws.PresignHandler)

	certFile := "/etc/letsencrypt/live/journeyapp.me/fullchain.pem"
	keyFile := "/etc/letsencrypt/live/journeyapp.me/privkey.pem"

	fmt.Println("Server running on port 443...")

	if err := http.ListenAndServeTLS(":443", certFile, keyFile, nil); err != nil {
		log.Fatalf("Failed to start TLS server: %v", err)
	}
}
