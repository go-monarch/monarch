package monarch

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ConnOptions func(*options.ClientOptions) error

type Connection struct {
	client *mongo.Client
}

type Monarch struct {
	conn       *Connection
	db         *mongo.Database
	cacheStore *sync.Map
}

func Connect(url string, opts ...ConnOptions) (*Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	options := options.Client()
	options = options.ApplyURI(url)
	options = options.SetRegistry(mongoRegistry)

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
	return &Monarch{conn: c, cacheStore: &sync.Map{}, db: c.client.Database("monarch")}
}

func (m *Monarch) UseDB(db string) {
	m.db = m.conn.client.Database(db)
}
