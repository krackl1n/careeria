package converter

import (
	"testing"
	"time"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	query "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancyquery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPageToken(t *testing.T) {
	companyID := uuid.New()
	cursor := &domain.Cursor{ID: uuid.New(), Time: time.Now().UTC().Truncate(time.Microsecond)}
	token, err := encodePageToken(cursor, companyID, query.ViewPublic)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodePageToken(token, companyID, query.ViewPublic)
	if err != nil || got.ID != cursor.ID || !got.Time.Equal(cursor.Time) {
		t.Fatalf("round trip: %+v, %v", got, err)
	}
	for _, tc := range []struct {
		token     string
		companyID uuid.UUID
		view      query.View
	}{
		{"invalid!", companyID, query.ViewPublic},
		{token, uuid.New(), query.ViewPublic},
		{token, companyID, query.ViewCompany},
	} {
		if _, err := decodePageToken(tc.token, tc.companyID, tc.view); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("invalid token: %v", err)
		}
	}
}
