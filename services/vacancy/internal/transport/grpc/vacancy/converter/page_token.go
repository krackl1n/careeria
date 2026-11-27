package converter

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	query "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancyquery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type pageToken struct {
	CompanyID uuid.UUID  `json:"company_id"`
	View      query.View `json:"view"`
	Time      time.Time  `json:"time"`
	ID        uuid.UUID  `json:"id"`
}

func decodePageToken(value string, companyID uuid.UUID, mode query.View) (*domain.Cursor, error) {
	if value == "" {
		return nil, nil
	}
	if len(value) > 2048 {
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}
	var token pageToken
	if err := json.Unmarshal(raw, &token); err != nil || token.CompanyID != companyID ||
		token.View != mode || token.Time.IsZero() || token.ID == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}
	return &domain.Cursor{Time: token.Time, ID: token.ID}, nil
}

func encodePageToken(cursor *domain.Cursor, companyID uuid.UUID, mode query.View) (string, error) {
	if cursor == nil {
		return "", nil
	}
	raw, err := json.Marshal(pageToken{CompanyID: companyID, View: mode, Time: cursor.Time, ID: cursor.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
