package grpcgateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type FormMarshaler struct {
	gatewayruntime.JSONPb
}

func (m FormMarshaler) NewDecoder(reader io.Reader) gatewayruntime.Decoder {
	return gatewayruntime.DecoderFunc(func(value any) error {
		message, ok := value.(proto.Message)
		if !ok {
			return fmt.Errorf("form target must be a protobuf message, got %T", value)
		}

		body, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("read form body: %w", err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return fmt.Errorf("parse form body: %w", err)
		}

		fields := make(map[string]any, len(values))
		for key, items := range values {
			if len(items) == 1 {
				fields[key] = items[0]
				continue
			}
			fields[key] = items
		}

		encoded, err := json.Marshal(fields)
		if err != nil {
			return fmt.Errorf("encode form fields: %w", err)
		}
		if err := (protojson.UnmarshalOptions{}).Unmarshal(encoded, message); err != nil {
			return fmt.Errorf("decode form fields: %w", err)
		}

		return nil
	})
}
