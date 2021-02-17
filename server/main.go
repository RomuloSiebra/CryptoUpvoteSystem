package main

import (
	"fmt"
	"log"
	"net"

	upvoteSystem "github.com/RomuloSiebra/CryptoUpvoteSystem/proto/UpvoteSystem"
	"google.golang.org/grpc"
)

type server struct {
	upvoteSystem.UnimplementedUpvoteSystemServer
}

func main() {
	port := 3333
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Server listening at port: %d \n", port)
	s := grpc.NewServer()
	upvoteSystem.RegisterUpvoteSystemServer(s, &server{})

	s.Serve(lis)

}
