package main

import (
	"context"
	"io"
	"log"
	"net/http"

	upvoteSystem "github.com/RomuloSiebra/CryptoUpvoteSystem/proto/UpvoteSystem"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:3333", grpc.WithInsecure())
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
				// log.Printf("Resp received: %s", resp.GetCrypto().Name)
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
	g.GET("/cryptoSumStream/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		request := &upvoteSystem.GetVoteSumStreamRequest{
			Id: id,
		}

		stream, err := client.GetVoteSumStream(ctx, request)
		if err != nil {
			ctx.JSON(http.StatusBadGateway, gin.H{"Error": "Couldn`t get vote sum"})
		}
		done := make(chan bool)

		go func() {
			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					done <- true
					return
				}
				if err != nil {
					done <- true
					ctx.JSON(http.StatusBadGateway, gin.H{"Error": "Couldn`t get vote sum"})
				}
				log.Printf("Resp received: %d \n", resp.GetVotes())

			}
		}()

		<-done
		log.Printf("finished")

	})
	// for i := 0; i < 10000; i++ {
	// 	go createLiveUpdate("6037662306086867038f7b1d", client)
	// }
	if err := g.Run(":8000"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func createLiveUpdate(id string, client upvoteSystem.UpvoteSystemClient) {
	request := &upvoteSystem.GetVoteSumStreamRequest{
		Id: id,
	}

	stream, err := client.GetVoteSumStream(context.Background(), request)
	if err != nil {
		log.Fatalf("open stream error %v", err)
	}
	done := make(chan bool)

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			if err != nil {
				log.Fatalf("cannot receive %v", err)
			}
			log.Printf("Resp received: %d \n", resp.GetVotes())

		}
	}()

	<-done
	log.Printf("finished")
}
