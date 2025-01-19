package db

import "go.mongodb.org/mongo-driver/mongo"

var MongoClient *mongo.Client
var DbName = "journeyDB"
var UserCollection = "users"
