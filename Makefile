generate: 
	@protoc --proto_path=$(shell pwd) --go_out=. --go-grpc_out=. proto/UpvoteSystem.proto

dev:
	@go run server/main.go

run-client:
	@go run client/main.go
	
test: 
	@go test -cover ./server

run-db:
	@docker run --rm -p 27017:27017 mongo

