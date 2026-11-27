package converter

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestInvalidAmount(t *testing.T) {
	for _, value := range []string{"-1.00", "1.234", "10000000000000000"} {
		if _, err := amount(&value); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("amount %s: %v", value, err)
		}
	}
}
