package main

import (
	"log"
	"net"

	pb "highloadgram/api"
	"highloadgram/internal/messenger"

	"google.golang.org/grpc"
)

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterMessengerServer(
		grpcServer,
		messenger.NewServer(),
	)

	log.Println("messenger listening on :50051")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
