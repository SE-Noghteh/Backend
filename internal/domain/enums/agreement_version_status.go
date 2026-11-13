package enums

import "database/sql/driver"

type AgreementVersionStatus uint

const (
	AgreementVersionStatusProposed   AgreementVersionStatus = iota + 1
	AgreementVersionStatusActive     AgreementVersionStatus = iota + 1
	AgreementVersionStatusSuperseded AgreementVersionStatus = iota + 1
)

func (value AgreementVersionStatus) String() string {
	switch value {
	case AgreementVersionStatusProposed:
		return "proposed"
	case AgreementVersionStatusActive:
		return "active"
	case AgreementVersionStatusSuperseded:
		return "superseded"
	}
	return ""
}

func GetAllAgreementVersionStatuses() []AgreementVersionStatus {
	return []AgreementVersionStatus{
		AgreementVersionStatusProposed,
		AgreementVersionStatusActive,
		AgreementVersionStatusSuperseded,
	}
}

func (value AgreementVersionStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *AgreementVersionStatus) Scan(src any) error {
	return scanEnum(src, map[string]AgreementVersionStatus{
		"proposed":   AgreementVersionStatusProposed,
		"active":     AgreementVersionStatusActive,
		"superseded": AgreementVersionStatusSuperseded,
	}, value)
}
