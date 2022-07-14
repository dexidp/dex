package mongo

import (
	"context"

	"github.com/dexidp/dex/storage"
	"github.com/pkg/errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const errElementDuplicatedCode = 11000

type Collection struct {
	collection *mongo.Collection
}

func (collection *Collection) Insert(c context.Context, data interface{}) error {
	_, err := collection.collection.InsertOne(c, data)
	if err != nil {
		if hasDuplicatedError(err) {
			return storage.ErrAlreadyExists
		}

		return err
	}

	return nil
}

func (collection *Collection) Find(c context.Context, dst interface{}, filter bson.D, findOptions ...*options.FindOptions) error {
	cursor, err := collection.collection.Find(c, filter, findOptions...)
	if err != nil {
		return err
	}

	defer cursor.Close(c)

	if err = cursor.All(c, dst); err != nil {
		return err
	}

	return nil
}

func (collection *Collection) FindOne(c context.Context, dst interface{}, filter bson.D, findOptions ...*options.FindOneOptions) error {
	result := collection.collection.FindOne(c, filter, findOptions...)
	if errors.Is(result.Err(), mongo.ErrNoDocuments) {
		return storage.ErrNotFound
	}

	err := result.Decode(dst)
	if err != nil {
		return err
	}

	return nil
}

func (collection *Collection) DeleteOne(c context.Context, filter bson.D) error {
	result, err := collection.collection.DeleteOne(c, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount < 1 {
		return storage.ErrNotFound
	}

	return nil
}

func (collection *Collection) DeleteMany(c context.Context, filter bson.D) (int64, error) {
	result, err := collection.collection.DeleteMany(c, filter)
	if err != nil {
		return 0, err
	}

	if result.DeletedCount == 0 {
		return 0, storage.ErrNotFound
	}

	return result.DeletedCount, nil
}

func (collection *Collection) UpdateOne(c context.Context, filter bson.D, toUpdate bson.D) error {
	result, err := collection.collection.UpdateOne(c, filter, toUpdate)
	if err != nil {
		if hasDuplicatedError(err) {
			return storage.ErrAlreadyExists
		}

		return err
	}

	if result.MatchedCount == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (collection *Collection) ReplaceOne(c context.Context, filter bson.D, toReplace interface{}) error {
	result, err := collection.collection.ReplaceOne(c, filter, toReplace)
	if err != nil {
		if hasDuplicatedError(err) {
			return storage.ErrAlreadyExists
		}
	}

	if result.MatchedCount < 1 {
		return storage.ErrNotFound
	}

	return nil
}

func (collection *Collection) CountDocuments(c context.Context, filter bson.D) (int64, error) {
	num, err := collection.collection.CountDocuments(c, filter)
	if err != nil {
		return 0, err
	}

	return num, nil
}

func (collection *Collection) AggregateDocument(c context.Context, dst interface{}, pipeline []bson.D) error {
	cursor, err := collection.collection.Aggregate(c, pipeline)
	if err != nil {
		return err
	}

	if err = cursor.All(c, dst); err != nil {
		return err
	}

	return nil
}

func (collection *Collection) CreateIndexes(c context.Context, indexes ...mongo.IndexModel) error {
	if _, err := collection.collection.Indexes().CreateMany(c, indexes); err != nil {
		return err
	}

	return nil
}

func hasDuplicatedError(err error) bool {
	var writeException mongo.WriteException

	if errors.As(err, &writeException) {
		for _, we := range writeException.WriteErrors {
			if we.Code == errElementDuplicatedCode {
				return true
			}
		}
	}

	return false
}
