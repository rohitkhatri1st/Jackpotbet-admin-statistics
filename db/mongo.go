package db

import (
	"admin-stats/config"
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoDB implements Database for a MongoDB connection.
// Access DB to pass *mongo.Database into repositories.
type MongoDB struct {
	Client *mongo.Client
	DB     *mongo.Database
	cfg    config.MongoConfig
}

func NewMongoDB(cfg config.MongoConfig) *MongoDB {
	return &MongoDB{cfg: cfg}
}

func (m *MongoDB) Connect(ctx context.Context) error {
	client, err := mongo.Connect(options.Client().ApplyURI(m.cfg.URI))
	if err != nil {
		return err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return err
	}
	m.Client = client
	m.DB = client.Database(m.cfg.DBName)
	return nil
}

func (m *MongoDB) Disconnect(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
