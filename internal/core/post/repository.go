package post

import (
	"context"
	"errors"
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
	return &Repository{
		coll: db.Collection("post"),
	}
}

func (r *Repository) Create(ctx context.Context, p *Post) (string, error) {
	now := time.Now()
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	p.IsRemove = false

	_, err := r.coll.InsertOne(ctx, p)
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

func (r *Repository) Update(ctx context.Context, id string, update bson.M) error {
	filter := bson.M{"_id": id, "isRemove": false}
	update["updatedAt"] = time.Now()
	res, err := r.coll.UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("not found")
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Post, error) {
	var p Post
	err := r.coll.FindOne(ctx, bson.M{"_id": id, "isRemove": false}).Decode(&p)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

type ListOptions struct {
	Limit  int64
	Skip   int64
	Sort   bson.D
	Filter bson.M
}

func (r *Repository) List(ctx context.Context, opts ListOptions) ([]Post, int64, error) {
	filter := bson.M{"isRemove": false}
	if opts.Filter != nil {
		for k, v := range opts.Filter {
			filter[k] = v
		}
	}
	findOpts := options.Find()
	if opts.Limit > 0 {
		findOpts.SetLimit(opts.Limit)
	}
	if opts.Skip > 0 {
		findOpts.SetSkip(opts.Skip)
	}
	if len(opts.Sort) > 0 {
		findOpts.SetSort(opts.Sort)
	}
	cur, err := r.coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var out []Post
	for cur.Next(ctx) {
		var p Post
		if err := cur.Decode(&p); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"isRemove": true, "updatedAt": time.Now()}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("not found")
	}
	return nil
}

func (r *Repository) RemoveSome(ctx context.Context, ids []string) error {
	_, err := r.coll.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": ids}}, bson.M{"$set": bson.M{"isRemove": true, "updatedAt": time.Now()}})
	return err
}

func (r *Repository) ToggleLike(ctx context.Context, postId, userId string) (bool, int, error) {
	var p Post
	if err := r.coll.FindOne(ctx, bson.M{"_id": postId}).Decode(&p); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, 0, errors.New("post not found")
		}
		return false, 0, err
	}

	liked := false
	likes := p.Like
	// find user
	idx := -1
	for i, u := range likes {
		if u == userId {
			idx = i
			break
		}
	}
	if idx == -1 {
		likes = append(likes, userId)
		liked = true
	} else {
		// remove
		likes = append(likes[:idx], likes[idx+1:]...)
		liked = false
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": postId}, bson.M{"$set": bson.M{"like": likes, "updatedAt": time.Now()}})
	if err != nil {
		return false, 0, err
	}
	return liked, len(likes), nil
}

func (r *Repository) ListWithComments(ctx context.Context, opts ListOptions) ([]Post, int64, error) {
	filter := bson.M{"isRemove": false}
	if opts.Filter != nil {
		for k, v := range opts.Filter {
			filter[k] = v
		}
	}

	// Pipeline del aggregate
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{
			{Key: "$lookup", Value: bson.M{
				"from":         "comment",
				"localField":   "_id",
				"foreignField": "relatedId",
				"as":           "comments",
			}},
		},
	}

	// Sort, skip, limit dinámicos
	if len(opts.Sort) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: opts.Sort}})
	}
	if opts.Skip > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: opts.Skip}})
	}
	if opts.Limit > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: opts.Limit}})
	}

	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var posts []Post
	for cur.Next(ctx) {
		var p Post
		if err := cur.Decode(&p); err != nil {
			return nil, 0, err
		}
		posts = append(posts, p)
	}
	if err := cur.Err(); err != nil {
		return nil, 0, err
	}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}
