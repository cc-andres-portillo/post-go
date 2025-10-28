package comment

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	coll *mongo.Collection
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{coll: db.Collection("comment")}
}

func (r *Repository) Create(ctx context.Context, c *Comment) (string, error) {
	now := time.Now()
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	c.IsRemove = false
	_, err := r.coll.InsertOne(ctx, c)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Comment, error) {
	var c Comment
	err := r.coll.FindOne(ctx, bson.M{"_id": id, "isRemove": false}).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListByRelated(ctx context.Context, relatedId string, limit int64) ([]Comment, int64, error) {
	filter := bson.M{"relatedId": relatedId, "isRemove": false}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opts.SetLimit(limit)
	}
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var out []Comment
	for cur.Next(ctx) {
		var c Comment
		if err := cur.Decode(&c); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *Repository) CountByRelated(ctx context.Context, relatedId string) (int64, error) {
	return r.coll.CountDocuments(ctx, bson.M{"relatedId": relatedId, "isRemove": false})
}

func (r *Repository) ListAllOrByRelated(ctx context.Context, relatedId string, limit, skip int64) ([]Comment, int64, error) {
	filter := bson.M{"isRemove": false}
	if relatedId != "" {
		filter["relatedId"] = relatedId
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		findOpts.SetLimit(limit)
	}
	if skip > 0 {
		findOpts.SetSkip(skip)
	}

	cur, err := r.coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var comments []Comment
	if err := cur.All(ctx, &comments); err != nil {
		return nil, 0, err
	}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}
