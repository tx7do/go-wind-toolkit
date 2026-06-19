package main

import (
	context "context"
	"encoding/json"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/menta2k/protoc-gen-redact/v3/examples/user/pb"
)

var errNotFound = status.Error(codes.NotFound, "User not found")

type core struct {
	pb.UnimplementedChatServer
	st map[string][]byte
}

func NewChatServer() pb.ChatServer {
	return &core{st: map[string][]byte{}}
}

func serialize(in *pb.User) ([]byte, error) { return json.Marshal(in) }
func deserialize(in []byte) (*pb.User, error) {
	out := &pb.User{}
	return out, json.Unmarshal(in, out)
}

func (c *core) AddUser(ctx context.Context, in *pb.User) (*pb.User, error) {
	_, err := c.GetUser(ctx, &pb.GetUserRequest{Username: in.Username})
	if err == errNotFound {
		// add user
		c.st[in.Username], err = serialize(in)
		return in, err
	}
	if err == nil {
		err = status.Error(codes.AlreadyExists, "User already exists")
	}
	return nil, err
}

func (c *core) GetUser(ctx context.Context, in *pb.GetUserRequest) (*pb.User, error) {
	if got, ok := c.st[in.Username]; ok {
		return deserialize(got)
	}
	return nil, errNotFound
}

func (c *core) ListUsers(ctx context.Context, in *empty.Empty) (*pb.ListUsersResponse, error) {
	res := &pb.ListUsersResponse{Users: make([]*pb.User, 0, len(c.st))}
	for _, val := range c.st {
		got, err := deserialize(val)
		if err != nil {
			return nil, err
		}
		res.Users = append(res.Users, got)
	}
	return res, nil
}
