package grpcgateway

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	grpchttpresponse "github.com/krackl1n/careeria/grpc/httpresponse"
	"google.golang.org/protobuf/proto"
)

func ApplyHTTPResponseMetadata(
	ctx context.Context,
	writer http.ResponseWriter,
	_ proto.Message,
) error {
	serverMetadata, ok := gatewayruntime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}

	statusValues := serverMetadata.HeaderMD.Get(grpchttpresponse.StatusCodeMetadataKey)
	if len(statusValues) == 0 {
		return nil
	}
	if len(statusValues) != 1 {
		return errors.New("HTTP response status metadata must contain exactly one value")
	}

	statusCode, err := strconv.Atoi(statusValues[0])
	if err != nil || statusCode < 200 || statusCode > 599 {
		return errors.New("HTTP response status metadata is invalid")
	}

	locationValues := serverMetadata.HeaderMD.Get(grpchttpresponse.LocationMetadataKey)
	if len(locationValues) > 1 {
		return errors.New("HTTP response location metadata must contain at most one value")
	}

	location := ""
	if len(locationValues) == 1 {
		location = locationValues[0]
		if strings.TrimSpace(location) == "" || strings.ContainsAny(location, "\r\n") {
			return errors.New("HTTP response location metadata is invalid")
		}
	}
	if statusCode >= 300 && statusCode <= 399 && location == "" {
		return errors.New("HTTP redirect response requires location metadata")
	}

	if location != "" {
		writer.Header().Set("Location", location)
	}
	writer.WriteHeader(statusCode)

	return nil
}
