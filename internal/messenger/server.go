package messenger

import (
	"context"
	"sync"
	"time"

	pb "highloadgram/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const clientQueueSize = 128

type Server struct {
	pb.UnimplementedMessengerServer

	mu      sync.RWMutex
	clients map[string]chan *pb.ChatMessage
}

func NewServer() *Server {
	return &Server{
		clients: make(map[string]chan *pb.ChatMessage),
	}
}

func (s *Server) Subscribe(
	req *pb.SubscribeRequest,
	stream grpc.ServerStreamingServer[pb.ChatMessage],
) error {
	if req.Username == "" {
		return status.Error(codes.InvalidArgument, "username is required")
	}

	messages := make(chan *pb.ChatMessage, clientQueueSize)

	s.mu.Lock()
	if _, exists := s.clients[req.Username]; exists {
		s.mu.Unlock()
		return status.Error(codes.AlreadyExists, "user is already connected")
	}

	s.clients[req.Username] = messages
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, req.Username)
		s.mu.Unlock()
	}()

	for {
		select {
		case msg := <-messages:
			if err := stream.Send(msg); err != nil {
				return err
			}

		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *Server) SendMessage(
	ctx context.Context,
	req *pb.SendMessageRequest,
) (*pb.SendMessageResponse, error) {
	if req.From == "" || req.To == "" || req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "from, to and text are required")
	}

	s.mu.RLock()
	messages, exists := s.clients[req.To]
	s.mu.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "recipient is offline")
	}

	msg := &pb.ChatMessage{
		From:       req.From,
		To:         req.To,
		Text:       req.Text,
		SentAtUnix: time.Now().Unix(),
	}

	select {
	case messages <- msg:
		return &pb.SendMessageResponse{
			Delivered: true,
		}, nil

	default:
		return nil, status.Error(codes.ResourceExhausted, "recipient queue is full")
	}
}
