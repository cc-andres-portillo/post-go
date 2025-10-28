package post

import (
	"posts/internal/core/comment"
	"time"

	"posts/pkg/utils"
)

type Post struct {
	ID                string                `bson:"_id" json:"_id"`
	Title             string                `bson:"title" json:"title"`
	UserId            string                `bson:"userId" json:"userId"`
	ImgUrl            utils.FlexibleStrings `bson:"imgUrl" json:"imgUrl"`
	Description       string                `bson:"description,omitempty" json:"description,omitempty"`
	Content           string                `bson:"content,omitempty" json:"content,omitempty"`
	PublishedId       string                `bson:"publishedId,omitempty" json:"publishedId,omitempty"`
	PublishedName     string                `bson:"publishedName,omitempty" json:"publishedName,omitempty"`
	Like              []string              `bson:"like,omitempty" json:"like,omitempty"`
	Status            bool                  `bson:"status" json:"status"`
	Fixed             bool                  `bson:"fixed,omitempty" json:"fixed,omitempty"`
	Privacy           bool                  `bson:"privacy" json:"privacy"`
	ImgHeight         string                `bson:"imgHeight,omitempty" json:"imgHeight,omitempty"`
	MediaCategory     string                `bson:"mediaCategory,omitempty" json:"mediaCategory,omitempty"`
	ExternalLinkTitle string                `bson:"externalLinkTitle,omitempty" json:"externalLinkTitle,omitempty"`
	PostGroupId       string                `bson:"postGroupId,omitempty" json:"postGroupId,omitempty"`
	Attachments       utils.FlexibleStrings `bson:"attachments,omitempty" json:"attachments,omitempty"`
	VideoCover        string                `bson:"videoCover,omitempty" json:"videoCover,omitempty"`
	IsRemove          bool                  `bson:"isRemove" json:"isRemove"`
	CreatedAt         time.Time             `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time             `bson:"updatedAt" json:"updatedAt"`

	Comments      []comment.Comment `bson:"comments,omitempty" json:"comments,omitempty"` // populated by $lookup
	TotalComments int64             `bson:"-" json:"totalComments,omitempty"`
}
