package mongo

import (
	"log"

	"go.mongodb.org/mongo-driver/mongo"
)

type mongoStorage struct {
	session  *Session
	database *mongo.Database

	config Mongo

	logger log.Logger
}
