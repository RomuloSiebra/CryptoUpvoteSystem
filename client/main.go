package main

import (
	"io"
	"log"
	"net/http"
	"os"

	upvoteSystem "github.com/RomuloSiebra/CryptoUpvoteSystem/proto/UpvoteSystem"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		log.Fatal("Error: Invalid SERVER_PORT environment variable")
	}

	clientPort := os.Getenv("CLIENT_PORT")
	if serverPort == "" {
		log.Fatal("Error: Invalid CLIENT_PORT environment variable")
	}

	conn, err := grpc.Dial("localhost:"+serverPort, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	client := upvoteSystem.NewUpvoteSystemClient(conn)
	g := gin.Default()

	g.POST("/crypto", func(ctx *gin.Context) {

		crypto := upvoteSystem.Cryptocurrency{}

		if err := ctx.ShouldBindJSON(&crypto); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid request body"})
			return
		}

		request := &upvoteSystem.CreateCryptoRequest{
			Crypto: &crypto,
		}

		result, err := client.CreateCrypto(ctx, request)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid request body"})
			return
		}
		ctx.JSON(http.StatusCreated, gin.H{
			"result": result,
		})

	})

	g.GET("/crypto", func(ctx *gin.Context) {
		request := &upvoteSystem.ReadAllCryptoRequest{}

		stream, err := client.ReadAllCrypto(ctx, request)
		if err != nil {
			ctx.JSON(http.StatusBadGateway, gin.H{"Error": "Couldn`t get cryptocurrencies"})
		}

		var result []*upvoteSystem.ReadAllCryptoResponse
		done := make(chan bool)

		go func() {
			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					done <- true
					return
				}
				if err != nil {
					ctx.JSON(http.StatusBadGateway, gin.H{"Error": "Couldn`t get cryptocurrencies"})
				}
				result = append(result, resp)

			}
		}()

		<-done
		ctx.JSON(http.StatusOK, gin.H{
			"result": result,
		})

	})

	g.GET("/crypto/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		request := &upvoteSystem.ReadCryptoByIDRequest{
			Id: id,
		}

		resp, err := client.ReadCryptoByID(ctx, request)
		if err != nil {
			ctx.JSON(404, gin.H{"Error": "Id not found"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"result": resp,
		})

	})

	g.DELETE("/crypto/:id", func(ctx *gin.Context) {

		id := ctx.Param("id")
		request := &upvoteSystem.DeleteCryptoRequest{
			Id: id,
		}

		resp, err := client.DeleteCrypto(ctx, request)
		if err != nil {
			ctx.JSON(404, gin.H{"Error": "Id not found"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"result": resp,
		})

	})
	g.PUT("/crypto", func(ctx *gin.Context) {
		crypto := upvoteSystem.Cryptocurrency{}

		if err := ctx.ShouldBindJSON(&crypto); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid request body"})
			return
		}

		request := &upvoteSystem.UpdateCryptoRequest{
			Crypto: &crypto,
		}

		result, err := client.UpdateCrypto(ctx, request)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid request body"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"result": result,
		})
	})

	g.POST("/crypto/upvote/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		request := &upvoteSystem.UpvoteCryptoRequest{
			Id: id,
		}

		resp, err := client.UpvoteCrypto(ctx, request)
		if err != nil {
			ctx.JSON(404, gin.H{"Error": "Id not found"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"result": resp,
		})

	})

	g.POST("/crypto/downvote/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		request := &upvoteSystem.DownvoteCryptoRequest{
			Id: id,
		}

		resp, err := client.DownvoteCrypto(ctx, request)
		if err != nil {
			ctx.JSON(404, gin.H{"Error": "Id not found"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"result": resp,
		})

	})
	g.GET("/cryptoSum/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		request := &upvoteSystem.GetVotesSumRequest{
			Id: id,
		}

		resp, err := client.GetVotesSum(ctx, request)
		if err != nil {
			ctx.JSON(404, gin.H{"Error": "Id not found"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"result": resp,
		})
	})

	if err := g.Run(":" + clientPort); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
