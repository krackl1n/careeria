package signingkeyadmin

import (
	"context"

	identityadminv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/admin/v1"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	CreateSigningKey(context.Context) (*signingkeydomain.SigningKey, error)
}

type Handler struct {
	identityadminv1.UnimplementedSigningKeyAdminServiceServer
	service Service
}

var _ identityadminv1.SigningKeyAdminServiceServer = (*Handler)(nil)

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateSigningKey(
	ctx context.Context,
	_ *identityadminv1.CreateSigningKeyRequest,
) (*identityadminv1.CreateSigningKeyResponse, error) {
	key, err := h.service.CreateSigningKey(ctx)
	if err != nil {
		return nil, err
	}
	return &identityadminv1.CreateSigningKeyResponse{SigningKey: toProto(key)}, nil
}

func toProto(key *signingkeydomain.SigningKey) *identityadminv1.SigningKey {
	result := &identityadminv1.SigningKey{
		Id:        key.ID.UUID().String(),
		KeyId:     string(key.KeyID),
		Algorithm: string(key.Algorithm),
		PublicJwk: string(key.PublicJWK),
		Status:    string(key.Status),
		CreatedAt: timestamppb.New(key.CreatedAt),
		NotBefore: timestamppb.New(key.NotBefore),
	}
	if key.ActivatedAt != nil {
		result.ActivatedAt = timestamppb.New(*key.ActivatedAt)
	}
	return result
}
