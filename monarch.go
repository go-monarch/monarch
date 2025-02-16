package monarch

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ConnOptions func(*options.ClientOptions) error

type Connection struct {
	client *mongo.Client
}

type Monarch struct {
	conn *Connection
	db   *mongo.Database
}

func Connect(url string, opts ...ConnOptions) (*Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	options := &options.ClientOptions{}
	options = options.ApplyURI(url)

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}
	client, err := mongo.Connect(options)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &Connection{client: client}, nil
}

func New(c *Connection) *Monarch {
	return &Monarch{conn: c}
}

func (m *Monarch) UseDB(db string) {
	m.db = m.conn.client.Database(db)
}
