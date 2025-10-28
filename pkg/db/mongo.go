package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoClient connects and pings MongoDB
func NewMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	opts := options.Client().ApplyURI(uri)
	ctxC, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctxC, opts)
	if err != nil {
		return nil, err
	}

	ctxP, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()
	if err := client.Ping(ctxP, nil); err != nil {
		return nil, err
	}

	log.Println("✅ MongoDB connected")
	return client, nil
}
