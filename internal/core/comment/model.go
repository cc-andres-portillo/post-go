package comment

import "time"

type Comment struct {
	ID        string    `bson:"_id" json:"_id"`
	UserId    string    `bson:"userId" json:"userId"`
	RelatedId string    `bson:"relatedId" json:"relatedId"`
	Message   string    `bson:"message" json:"message"`
	IsRemove  bool      `bson:"isRemove" json:"isRemove"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}
