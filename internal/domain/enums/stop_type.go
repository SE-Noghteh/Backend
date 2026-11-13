package enums

import "database/sql/driver"

type StopType uint

const (
	StopTypePause      StopType = iota + 1
	StopTypeCancel     StopType = iota + 1
	StopTypeNoResponse StopType = iota + 1
	StopTypeEarlyExit  StopType = iota + 1
)

func (value StopType) String() string {
	switch value {
	case StopTypePause:
		return "pause"
	case StopTypeCancel:
		return "cancel"
	case StopTypeNoResponse:
		return "no_response"
	case StopTypeEarlyExit:
		return "early_exit"
	}
	return ""
}

func GetAllStopTypes() []StopType {
	return []StopType{
		StopTypePause,
		StopTypeCancel,
		StopTypeNoResponse,
		StopTypeEarlyExit,
	}
}

func (value StopType) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *StopType) Scan(src any) error {
	return scanEnum(src, map[string]StopType{
		"pause":       StopTypePause,
		"cancel":      StopTypeCancel,
		"no_response": StopTypeNoResponse,
		"early_exit":  StopTypeEarlyExit,
	}, value)
}
