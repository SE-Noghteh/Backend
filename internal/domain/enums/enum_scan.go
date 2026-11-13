package enums

import (
	"fmt"
)

func scanEnum[T ~uint](src any, values map[string]T, target *T) error {
	var label string
	switch value := src.(type) {
	case string:
		label = value
	case []byte:
		label = string(value)
	case nil:
		return fmt.Errorf("cannot scan NULL into enum")
	default:
		return fmt.Errorf("cannot scan %T into enum", src)
	}
	parsed, ok := values[label]
	if !ok {
		return fmt.Errorf("unknown enum value %q", label)
	}
	*target = parsed
	return nil
}
