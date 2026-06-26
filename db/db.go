package db

import "context"

// Database is the swappable connection interface.
// To add a new database (e.g. Redis), implement this interface.
// To replace MongoDB, write a new struct and swap it in server.go — no service code changes.
type Database interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
}
