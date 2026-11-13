package enums

import "database/sql/driver"

type OfferStatus uint

const (
	OfferStatusSent     OfferStatus = iota + 1
	OfferStatusAccepted OfferStatus = iota + 1
	OfferStatusDeclined OfferStatus = iota + 1
	OfferStatusExpired  OfferStatus = iota + 1
)

func (value OfferStatus) String() string {
	switch value {
	case OfferStatusSent:
		return "sent"
	case OfferStatusAccepted:
		return "accepted"
	case OfferStatusDeclined:
		return "declined"
	case OfferStatusExpired:
		return "expired"
	}
	return ""
}

func GetAllOfferStatuses() []OfferStatus {
	return []OfferStatus{
		OfferStatusSent,
		OfferStatusAccepted,
		OfferStatusDeclined,
		OfferStatusExpired,
	}
}

func (value OfferStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *OfferStatus) Scan(src any) error {
	return scanEnum(src, map[string]OfferStatus{
		"sent":     OfferStatusSent,
		"accepted": OfferStatusAccepted,
		"declined": OfferStatusDeclined,
		"expired":  OfferStatusExpired,
	}, value)
}
