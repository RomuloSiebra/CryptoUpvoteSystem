package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"go.mongodb.org/mongo-driver/mongo/options"

	upvoteSystem "github.com/RomuloSiebra/CryptoUpvoteSystem/proto/UpvoteSystem"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
)

var dbClient *mongo.Client
var mongoCtx context.Context

type server struct {
	upvoteSystem.UnimplementedUpvoteSystemServer
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

	fmt.Println("Connected to MongoDB")

	serverPort := 3333
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Server listening at port: %d \n", serverPort)
	s := grpc.NewServer()
	upvoteSystem.RegisterUpvoteSystemServer(s, &server{})

	s.Serve(lis)

}
