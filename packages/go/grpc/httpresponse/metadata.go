package httpresponse

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	StatusCodeMetadataKey = "x-http-code"
	LocationMetadataKey   = "location"
)

func SetRedirect(ctx context.Context, statusCode int, location string) error {
	if statusCode < 300 || statusCode > 399 {
		return errors.New("redirect HTTP status code must be in the 3xx range")
	}
	if strings.TrimSpace(location) == "" {
		return errors.New("redirect location is required")
	}

	return grpc.SetHeader(ctx, metadata.Pairs(
		StatusCodeMetadataKey, strconv.Itoa(statusCode),
		LocationMetadataKey, location,
	))
}
