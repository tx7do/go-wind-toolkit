package sqlproto

import (
	"context"
	"testing"
)

func TestConverter(t *testing.T) {
	ctx := context.Background()

	dsn := "postgres://postgres:*Abcd123456@localhost:5432/example?sslmode=disable"
	moduleName := "admin"
	sourceModuleName := "user"
	moduleVersion := "v1"
	serviceType := "grpc"
	outputPath := "./api/protos"
	_, _ = Convert(
		ctx,
		&dsn,
		&outputPath,
		&moduleName, &sourceModuleName, &moduleVersion,
		&serviceType,
		nil, nil,
		true,
	)
}
