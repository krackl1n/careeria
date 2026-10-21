package offer

type OfferStatus string

const (
	StatusDraft    OfferStatus = "DRAFT"
	StatusSent     OfferStatus = "SENT"
	StatusAccepted OfferStatus = "ACCEPTED"
	StatusDeclined OfferStatus = "DECLINED"
	StatusExpired  OfferStatus = "EXPIRED"
	StatusRevoked  OfferStatus = "REVOKED"
)

func (s OfferStatus) String() string {
	return string(s)
}

func (s OfferStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusSent, StatusAccepted, StatusDeclined, StatusExpired, StatusRevoked:
		return true
	default:
		return false
	}
}

func (s OfferStatus) IsTerminal() bool {
	switch s {
	case StatusAccepted, StatusDeclined, StatusExpired, StatusRevoked:
		return true
	default:
		return false
	}
}

