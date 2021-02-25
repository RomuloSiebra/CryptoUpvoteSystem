package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	upvoteSystem "github.com/RomuloSiebra/CryptoUpvoteSystem/proto/UpvoteSystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func dialer() func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)

	s := grpc.NewServer()

	upvoteSystem.RegisterUpvoteSystemServer(s, &server{})

	go func() {
		if err := s.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func setupDB() {
	dbClient, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))

	if err != nil {
		log.Fatal(err)
	}

	mongoCtx = context.Background()
	err = dbClient.Connect(mongoCtx)

	if err != nil {
		log.Fatal(err)
	}

	db = dbClient.Database("UpvoteSystemTest").Collection("Cryptocurrency")
	fmt.Println("Connected to MongoDB")
}

func clearDB() {
	db.Drop(mongoCtx)
}

func TestCreateCrypto(t *testing.T) {
	setupDB()
	defer clearDB()
	grpcServer := server{}

	// Test request with empty fields
	emptyRequest := &upvoteSystem.CreateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Name:        "",
			Description: "",
		},
	}

	response, err := grpcServer.CreateCrypto(context.Background(), emptyRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = InvalidArgument desc = Empty fields", err.Error())

	// Test with valid request
	validRequest := &upvoteSystem.CreateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
	}

	response, err = grpcServer.CreateCrypto(context.Background(), validRequest)

	require.Nil(t, err)

	assert.Equal(t, "Bitcoin", response.GetCrypto().GetName())
	assert.Equal(t, "The most valuable cryptocurrency", response.GetCrypto().GetDescription())
	assert.Equal(t, int32(0), response.GetCrypto().GetUpvote())
	assert.Equal(t, int32(0), response.GetCrypto().GetDownvote())

	// Test with valid request but crypto already exists
	response, err = grpcServer.CreateCrypto(context.Background(), validRequest)

	require.NotNil(t, err)
	assert.Equal(t, "rpc error: code = AlreadyExists desc = Cryptocurrency already exists", err.Error())

}

func TestReadCryptoByID(t *testing.T) {
	setupDB()
	defer clearDB()
	grpcServer := server{}

	// Test request with empty ID
	emptyIDRequest := &upvoteSystem.ReadCryptoByIDRequest{
		Id: "",
	}
	response, err := grpcServer.ReadCryptoByID(context.Background(), emptyIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = InvalidArgument desc = the provided hex string is not a valid ObjectID", err.Error())

	// Test request with valid ID but not found on DB
	NotFoundIDRequest := &upvoteSystem.ReadCryptoByIDRequest{
		Id: primitive.NewObjectID().Hex(),
	}

	response, err = grpcServer.ReadCryptoByID(context.Background(), NotFoundIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = NotFound desc = Couldn`t find Cryptocurrency with Object Id", err.Error())

	// Test with valid request
	createRequest := &upvoteSystem.CreateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
	}

	cryptoResponse, err := grpcServer.CreateCrypto(context.Background(), createRequest)

	require.Nil(t, err)

	validRequest := &upvoteSystem.ReadCryptoByIDRequest{
		Id: cryptoResponse.Crypto.GetId(),
	}

	response, err = grpcServer.ReadCryptoByID(context.Background(), validRequest)

	require.Nil(t, err)

	assert.Equal(t, cryptoResponse.GetCrypto().GetId(), response.GetCrypto().GetId())
	assert.Equal(t, cryptoResponse.GetCrypto().GetName(), response.GetCrypto().GetName())
	assert.Equal(t, cryptoResponse.GetCrypto().GetDescription(), response.GetCrypto().GetDescription())
	assert.Equal(t, cryptoResponse.GetCrypto().GetUpvote(), response.GetCrypto().GetUpvote())
	assert.Equal(t, cryptoResponse.GetCrypto().GetDownvote(), response.GetCrypto().GetDownvote())

}

func TestReadAllCrypto(t *testing.T) {
	setupDB()
	defer clearDB()
	grpcServer := server{}

	// Test with no crypto found on DB
	var allCreatedCrypto []*upvoteSystem.CreateCryptoResponse

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := upvoteSystem.NewUpvoteSystemClient(conn)
	request := &upvoteSystem.ReadAllCryptoRequest{}

	stream, err := client.ReadAllCrypto(ctx, request)

	var result []*upvoteSystem.ReadAllCryptoResponse
	done := make(chan bool)

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			require.Nil(t, err)

			result = append(result, resp)

		}
	}()

	<-done

	require.Nil(t, err)
	require.Nil(t, result)

	// Test with populated DB
	cryptoTest := []*upvoteSystem.Cryptocurrency{
		&upvoteSystem.Cryptocurrency{
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
		&upvoteSystem.Cryptocurrency{
			Name:        "Ethereum",
			Description: "Second-largest cryptocurrency by market capitalization",
		},
		&upvoteSystem.Cryptocurrency{
			Name:        "Dogecoin",
			Description: "Elon`s favorite cryptocurrency",
		},
	}

	for _, crypto := range cryptoTest {
		createRequest := &upvoteSystem.CreateCryptoRequest{Crypto: crypto}
		response, err := grpcServer.CreateCrypto(context.Background(), createRequest)
		require.Nil(t, err)
		allCreatedCrypto = append(allCreatedCrypto, response)
	}

	stream, err = client.ReadAllCrypto(ctx, request)

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			require.Nil(t, err)

			result = append(result, resp)

		}
	}()

	<-done

	require.Nil(t, err)
	for i, crypto := range cryptoTest {
		assert.Equal(t, crypto.GetId(), result[i].GetCrypto().GetId())
		assert.Equal(t, crypto.GetName(), result[i].GetCrypto().GetName())
		assert.Equal(t, crypto.GetDescription(), result[i].GetCrypto().GetDescription())
		assert.Equal(t, crypto.GetUpvote(), result[i].GetCrypto().GetUpvote())
		assert.Equal(t, crypto.GetDownvote(), result[i].GetCrypto().GetDownvote())
	}

}

func TestDeleteCrypto(t *testing.T) {
	setupDB()
	defer clearDB()
	grpcServer := server{}

	// Test request with empty ID
	emptyIDRequest := &upvoteSystem.DeleteCryptoRequest{
		Id: "",
	}
	_, err := grpcServer.DeleteCrypto(context.Background(), emptyIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = InvalidArgument desc = the provided hex string is not a valid ObjectID", err.Error())

	// Test request with valid ID but not found on DB
	NotFoundIDRequest := &upvoteSystem.DeleteCryptoRequest{
		Id: primitive.NewObjectID().Hex(),
	}

	_, err = grpcServer.DeleteCrypto(context.Background(), NotFoundIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = NotFound desc = Couldn`t find Cryptocurrency with Object Id", err.Error())

	// Test with valid request
	createRequest := &upvoteSystem.CreateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
	}

	cryptoResponse, err := grpcServer.CreateCrypto(context.Background(), createRequest)

	require.Nil(t, err)

	validRequest := &upvoteSystem.DeleteCryptoRequest{
		Id: cryptoResponse.Crypto.GetId(),
	}

	response, err := grpcServer.DeleteCrypto(context.Background(), validRequest)

	require.Nil(t, err)

	assert.Equal(t, true, response.GetSuccess())

}

func TestUpdateCrypto(t *testing.T) {
	setupDB()
	defer clearDB()
	grpcServer := server{}

	// Test request with empty ID
	emptyIDRequest := &upvoteSystem.UpdateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Id: "",
		},
	}
	_, err := grpcServer.UpdateCrypto(context.Background(), emptyIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = InvalidArgument desc = the provided hex string is not a valid ObjectID", err.Error())

	// Test request with empty fields
	emptyFieldsRequest := &upvoteSystem.UpdateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Id:          primitive.NewObjectID().Hex(),
			Name:        "",
			Description: "",
		},
	}
	_, err = grpcServer.UpdateCrypto(context.Background(), emptyFieldsRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = InvalidArgument desc = Empty fields", err.Error())

	// Test with valid request but ID not found on DB
	NotFoundIDRequest := &upvoteSystem.UpdateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Id:          primitive.NewObjectID().Hex(),
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
	}
	_, err = grpcServer.UpdateCrypto(context.Background(), NotFoundIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = NotFound desc = Couldn`t find Cryptocurrency with Object Id", err.Error())

	// Test with valid request
	createRequest := &upvoteSystem.CreateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
	}

	cryptoResponse, err := grpcServer.CreateCrypto(context.Background(), createRequest)

	require.Nil(t, err)

	newName := "Ethereum"
	newDescription := "Second-largest cryptocurrency by market capitalization"
	validRequest := &upvoteSystem.UpdateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Id:          cryptoResponse.GetCrypto().GetId(),
			Name:        newName,
			Description: newDescription,
		},
	}

	response, err := grpcServer.UpdateCrypto(context.Background(), validRequest)

	require.Nil(t, err)

	assert.Equal(t, cryptoResponse.GetCrypto().GetId(), response.GetCrypto().GetId())
	assert.Equal(t, newName, response.GetCrypto().GetName())
	assert.Equal(t, newDescription, response.GetCrypto().GetDescription())
	assert.Equal(t, cryptoResponse.GetCrypto().GetUpvote(), response.GetCrypto().GetUpvote())
	assert.Equal(t, cryptoResponse.GetCrypto().GetDownvote(), response.GetCrypto().GetDownvote())

}

func TestUpvoteCrypto(t *testing.T) {
	setupDB()
	defer clearDB()
	grpcServer := server{}

	// Test request with empty ID
	emptyIDRequest := &upvoteSystem.UpvoteCryptoRequest{
		Id: "",
	}
	_, err := grpcServer.UpvoteCrypto(context.Background(), emptyIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = InvalidArgument desc = the provided hex string is not a valid ObjectID", err.Error())

	// Test request with valid ID but not found on DB
	NotFoundIDRequest := &upvoteSystem.UpvoteCryptoRequest{
		Id: primitive.NewObjectID().Hex(),
	}
	_, err = grpcServer.UpvoteCrypto(context.Background(), NotFoundIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = NotFound desc = Couldn`t find Cryptocurrency with Object Id", err.Error())

	// Test with valid request
	createRequest := &upvoteSystem.CreateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
	}

	cryptoResponse, err := grpcServer.CreateCrypto(context.Background(), createRequest)

	require.Nil(t, err)

	validRequest := &upvoteSystem.UpvoteCryptoRequest{
		Id: cryptoResponse.GetCrypto().GetId(),
	}
	response, err := grpcServer.UpvoteCrypto(context.Background(), validRequest)

	require.Nil(t, err)

	assert.Equal(t, cryptoResponse.GetCrypto().GetId(), response.GetCrypto().GetId())
	assert.Equal(t, cryptoResponse.GetCrypto().GetName(), response.GetCrypto().GetName())
	assert.Equal(t, cryptoResponse.GetCrypto().GetDescription(), response.GetCrypto().GetDescription())
	assert.Equal(t, cryptoResponse.GetCrypto().GetUpvote()+1, response.GetCrypto().GetUpvote())
	assert.Equal(t, cryptoResponse.GetCrypto().GetDownvote(), response.GetCrypto().GetDownvote())

}

func TestDownvoteCrypto(t *testing.T) {
	setupDB()
	defer clearDB()
	grpcServer := server{}

	// Test request with empty ID
	emptyIDRequest := &upvoteSystem.DownvoteCryptoRequest{
		Id: "",
	}
	_, err := grpcServer.DownvoteCrypto(context.Background(), emptyIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = InvalidArgument desc = the provided hex string is not a valid ObjectID", err.Error())

	// Test request with valid ID but not found on DB
	NotFoundIDRequest := &upvoteSystem.DownvoteCryptoRequest{
		Id: primitive.NewObjectID().Hex(),
	}
	_, err = grpcServer.DownvoteCrypto(context.Background(), NotFoundIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = NotFound desc = Couldn`t find Cryptocurrency with Object Id", err.Error())

	// Test with valid request
	createRequest := &upvoteSystem.CreateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
	}

	cryptoResponse, err := grpcServer.CreateCrypto(context.Background(), createRequest)

	require.Nil(t, err)

	validRequest := &upvoteSystem.DownvoteCryptoRequest{
		Id: cryptoResponse.GetCrypto().GetId(),
	}
	response, err := grpcServer.DownvoteCrypto(context.Background(), validRequest)

	require.Nil(t, err)

	assert.Equal(t, cryptoResponse.GetCrypto().GetId(), response.GetCrypto().GetId())
	assert.Equal(t, cryptoResponse.GetCrypto().GetName(), response.GetCrypto().GetName())
	assert.Equal(t, cryptoResponse.GetCrypto().GetDescription(), response.GetCrypto().GetDescription())
	assert.Equal(t, cryptoResponse.GetCrypto().GetUpvote(), response.GetCrypto().GetUpvote())
	assert.Equal(t, cryptoResponse.GetCrypto().GetDownvote()+1, response.GetCrypto().GetDownvote())

}

func TestGetVotesSum(t *testing.T) {
	setupDB()
	defer clearDB()
	grpcServer := server{}

	// Test request with empty ID
	emptyIDRequest := &upvoteSystem.GetVotesSumRequest{
		Id: "",
	}
	_, err := grpcServer.GetVotesSum(context.Background(), emptyIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = InvalidArgument desc = the provided hex string is not a valid ObjectID", err.Error())

	// Test request with valid ID but not found on DB
	NotFoundIDRequest := &upvoteSystem.GetVotesSumRequest{
		Id: primitive.NewObjectID().Hex(),
	}
	_, err = grpcServer.GetVotesSum(context.Background(), NotFoundIDRequest)

	require.NotNil(t, err)

	assert.Equal(t, "rpc error: code = NotFound desc = Couldn`t find Cryptocurrency with Object Id", err.Error())

	// Test with valid request
	createRequest := &upvoteSystem.CreateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
	}

	cryptoResponse, err := grpcServer.CreateCrypto(context.Background(), createRequest)

	require.Nil(t, err)

	var downvoteResponse *upvoteSystem.DownvoteCryptoResponse

	for i := 0; i < 5; i++ {
		validRequest := &upvoteSystem.UpvoteCryptoRequest{
			Id: cryptoResponse.GetCrypto().GetId(),
		}
		_, err = grpcServer.UpvoteCrypto(context.Background(), validRequest)

		require.Nil(t, err)
	}

	for i := 0; i < 2; i++ {
		validRequest := &upvoteSystem.DownvoteCryptoRequest{
			Id: cryptoResponse.GetCrypto().GetId(),
		}
		downvoteResponse, err = grpcServer.DownvoteCrypto(context.Background(), validRequest)

		require.Nil(t, err)
	}

	validRequest := &upvoteSystem.GetVotesSumRequest{
		Id: cryptoResponse.GetCrypto().GetId(),
	}
	response, err := grpcServer.GetVotesSum(context.Background(), validRequest)

	require.Nil(t, err)

	assert.Equal(t, downvoteResponse.GetCrypto().GetUpvote()-downvoteResponse.GetCrypto().GetDownvote(), response.GetVotes())

}

func TestGetVotesSumStream(t *testing.T) {
	setupDB()
	defer clearDB()
	grpcServer := server{}
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Test invalid request
	client := upvoteSystem.NewUpvoteSystemClient(conn)
	request := &upvoteSystem.GetVoteSumStreamRequest{
		Id: "",
	}

	stream, err := client.GetVoteSumStream(ctx, request)

	var result []*upvoteSystem.GetVoteSumStreamResponse
	var resp *upvoteSystem.GetVoteSumStreamResponse

	done := make(chan bool)

	go func() {
		for {
			resp, err = stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			if err != nil {
				done <- true
				return
			}

			result = append(result, resp)

		}
	}()

	<-done
	require.NotNil(t, err)
	require.Nil(t, result)

	// Test valid live update
	createRequest := &upvoteSystem.CreateCryptoRequest{
		Crypto: &upvoteSystem.Cryptocurrency{
			Name:        "Bitcoin",
			Description: "The most valuable cryptocurrency",
		},
	}

	cryptoResponse, err := grpcServer.CreateCrypto(context.Background(), createRequest)

	require.Nil(t, err)

	request = &upvoteSystem.GetVoteSumStreamRequest{
		Id: cryptoResponse.GetCrypto().GetId(),
	}

	stream, err = client.GetVoteSumStream(ctx, request)

	go func() {
		for {
			resp, err = stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			if err != nil {
				done <- true
				return
			}

		}
	}()

	go func() {

		for i := 0; i < 4; i++ {
			upvoteRequest := &upvoteSystem.UpvoteCryptoRequest{
				Id: cryptoResponse.GetCrypto().GetId(),
			}
			_, err = grpcServer.UpvoteCrypto(context.Background(), upvoteRequest)

			require.Nil(t, err)

		}

		for i := 0; i < 2; i++ {
			downvoteRequest := &upvoteSystem.DownvoteCryptoRequest{
				Id: cryptoResponse.GetCrypto().GetId(),
			}
			_, err = grpcServer.DownvoteCrypto(context.Background(), downvoteRequest)

			require.Nil(t, err)

		}
		// Sleep to give stream time to receive before finish Test
		time.Sleep(200 * time.Microsecond)
		done <- true
	}()

	<-done

	require.Nil(t, err)

	require.Equal(t, int32(2), resp.GetVotes())

}
