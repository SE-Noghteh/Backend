package enums

import "database/sql/driver"

type ReportKind uint

const (
	ReportKindSafety ReportKind = iota + 1
	ReportKindOther  ReportKind = iota + 1
)

func (value ReportKind) String() string {
	switch value {
	case ReportKindSafety:
		return "safety"
	case ReportKindOther:
		return "other"
	}
	return ""
}

func GetAllReportKinds() []ReportKind {
	return []ReportKind{
		ReportKindSafety,
		ReportKindOther,
	}
}

func (value ReportKind) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *ReportKind) Scan(src any) error {
	return scanEnum(src, map[string]ReportKind{
		"safety": ReportKindSafety,
		"other":  ReportKindOther,
	}, value)
}
