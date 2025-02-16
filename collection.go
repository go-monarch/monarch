package monarch

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Collection[T any] struct {
	schema *Schema
	coll   *mongo.Collection
	query  struct {
		filter bson.D
		sort   bson.D
	}
	Model[T]
}

type Model[T any] interface {
	Query() Querier[T]
	ExecRaw(ctx context.Context, f func(coll *Collection[T]) error) (*T, error)
	Saver[T]
}

type Querier[T any] interface {
	FindOne() (*T, error)
	FindMany() ([]*T, error)
	UpdateOne(ctx context.Context, doc T) error
	UpdateMany(ctx context.Context, doc T) error
	DeleteOne(ctx context.Context, doc T) error
	DeleteMany(ctx context.Context, doc T) error
}

type Saver[T any] interface {
	Save(ctx context.Context, doc T) error
}

func RegisterCollection[T any](m *Monarch, schema T) (*Collection[T], error) {
	s, err := ParseSchema(schema)
	if err != nil {
		return nil, err
	}

	coll := m.db.Collection(s.Collection)
	if err := registerIndexes(coll, s.Fields); err != nil {
		return nil, err
	}
	c := &Collection[T]{schema: s, coll: coll}

	return c, nil
}

func (c *Collection[T]) Query() Querier[T] {
	return c
}

func (c *Collection[T]) Save(ctx context.Context, doc T) error {
	bsonValue := convertToBson(doc)
	if _, err := c.coll.InsertOne(ctx, bsonValue); err != nil {
		return err
	}
	return nil
}
func (c *Collection[T]) FindOne() (*T, error) {
	return nil, nil
}
func (c *Collection[T]) FindMany() ([]*T, error) {
	return nil, nil
}

func (c *Collection[T]) UpdateOne(ctx context.Context, doc T) error {
	return nil
}
func (c *Collection[T]) UpdateMany(ctx context.Context, doc T) error {
	return nil
}
func (c *Collection[T]) DeleteOne(ctx context.Context, doc T) error {
	return nil
}
func (c *Collection[T]) DeleteMany(ctx context.Context, doc T) error {
	return nil
}

func (c *Collection[T]) Collection() *mongo.Collection {
	return c.coll
}

func registerIndexes(coll *mongo.Collection, fields []*Field) error {
	var idx []mongo.IndexModel
	for _, field := range fields {
		if field.Index {
			idx = append(idx, mongo.IndexModel{
				Keys:    bson.D{{Key: fmt.Sprintf("idx_%v", field.DBName), Value: 1}},
				Options: options.Index().SetUnique(true),
			})
		}
	}

	if len(idx) < 1 {
		return nil
	}
	_, err := coll.Indexes().CreateMany(context.Background(), idx)
	if err != nil {
		return err
	}

	return nil
}

func convertToBson(v interface{}) bool { return true }
