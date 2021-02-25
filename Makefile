generate: 
	@protoc --proto_path=$(shell pwd) --go_out=. --go-grpc_out=. proto/UpvoteSystem.proto

dev:
	@go run server/main.go

test: 
	@go test -cover ./server

