package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// Crypto - Cryptocurrency MongoDB model
type Crypto struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Upvote      int32              `json:"upvote" bson:"upvote"`
	Downvote    int32              `json:"downvote" bson:"downvote"`
}
