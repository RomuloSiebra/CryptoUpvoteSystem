package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo/options"

	model "github.com/RomuloSiebra/CryptoUpvoteSystem/model"
	upvoteSystem "github.com/RomuloSiebra/CryptoUpvoteSystem/proto/UpvoteSystem"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
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

func (*server) ReadCryptoByID(ctx context.Context, request *upvoteSystem.ReadCryptoByIDRequest) (*upvoteSystem.ReadCryptoByIDResponse, error) {
	cryptoID, err := primitive.ObjectIDFromHex(request.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Error: %v", err))
	}

	result := db.FindOne(mongoCtx, bson.M{"_id": cryptoID})

	data := model.Crypto{}

	if err := result.Decode(&data); err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Couldn`t find Cryptocurrency with Object Id %s: %v", request.GetId(), err))
	}

	response := &upvoteSystem.ReadCryptoByIDResponse{
		Crypto: &upvoteSystem.Cryptocurrency{
			Id:          data.ID.Hex(),
			Name:        data.Name,
			Description: data.Description,
			Downvote:    data.Downvote,
			Upvote:      data.Upvote,
		},
	}
	return response, nil
}

func (*server) ReadAllCrypto(request *upvoteSystem.ReadAllCryptoRequest, stream upvoteSystem.UpvoteSystem_ReadAllCryptoServer) error {
	data := &model.Crypto{}

	pointer, err := db.Find(mongoCtx, bson.M{})
	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Error: %v", err))
	}

	defer pointer.Close(mongoCtx)

	for pointer.Next(mongoCtx) {
		err := pointer.Decode(data)
		if err != nil {
			return status.Errorf(codes.Unavailable, fmt.Sprintf("Couldn`t decode data: %v", err))
		}

		stream.Send(&upvoteSystem.ReadAllCryptoResponse{
			Crypto: &upvoteSystem.Cryptocurrency{
				Id:          data.ID.Hex(),
				Name:        data.Name,
				Description: data.Description,
				Downvote:    data.Downvote,
				Upvote:      data.Upvote,
			},
		})
	}
	if err := pointer.Err(); err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unkown mongoDB pointer error: %v", err))
	}
	return nil
}

func (*server) DeleteCrypto(ctx context.Context, request *upvoteSystem.DeleteCryptoRequest) (*upvoteSystem.DeleteCryptoResponse, error) {
	cryptoID, err := primitive.ObjectIDFromHex(request.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Error: %v", err))
	}

	_, err = db.DeleteOne(mongoCtx, bson.M{"_id": cryptoID})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Couldn`t delete Cryptocurrency with id %s: %v", request.GetId(), err))
	}

	response := &upvoteSystem.DeleteCryptoResponse{
		Success: true,
	}
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
