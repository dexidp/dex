package mongo

import (
	"log"

	"go.mongodb.org/mongo-driver/mongo"
)

type mongoStorage struct {
	database *mongo.Database

	config Mongo

	logger log.Logger
}
