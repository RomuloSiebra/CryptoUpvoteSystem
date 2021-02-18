package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo/options"

	model "github.com/RomuloSiebra/CryptoUpvoteSystem/model"
	upvoteSystem "github.com/RomuloSiebra/CryptoUpvoteSystem/proto/UpvoteSystem"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var dbClient *mongo.Client
var mongoCtx context.Context
var db *mongo.Collection

type server struct {
	upvoteSystem.UnimplementedUpvoteSystemServer
}

func (*server) CreateCrypto(ctx context.Context, request *upvoteSystem.CreateCryptoRequest) (*upvoteSystem.CreateCryptoResponse, error) {
	crypto := request.GetCrypto()

	data := model.Crypto{
		ID:          primitive.NewObjectID(),
		Name:        crypto.GetName(),
		Description: crypto.GetDescription(),
		Upvote:      0,
		Downvote:    0,
	}

	insertResult, err := db.InsertOne(mongoCtx, data)
	if err != nil {
		log.Fatal(err)
	}

	crypto.Id = insertResult.InsertedID.(primitive.ObjectID).Hex()
	crypto.Upvote = 0
	crypto.Downvote = 0

	response := &upvoteSystem.CreateCryptoResponse{Crypto: crypto}

	return response, nil
}

func main() {
	dbClient, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))

	if err != nil {
		log.Fatal(err)
	}

	mongoCtx = context.Background()
	err = dbClient.Connect(mongoCtx)

	if err != nil {
		log.Fatal(err)
	}

	db = dbClient.Database("UpvoteSystem").Collection("Cryptocurrency")
	fmt.Println("Connected to MongoDB")

	serverPort := 3333
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Server listening at port: %d \n", serverPort)
	s := grpc.NewServer()

	reflection.Register(s)
	upvoteSystem.RegisterUpvoteSystemServer(s, &server{})

	s.Serve(lis)

}
