package database

import (
	"log"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func NewClient(uri string) *mongo.Client {

	if uri == "" {
		log.Fatal("Set the 'MONGODB_URI' environment variable")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	return client
}
