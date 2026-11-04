package domainerrors

import "errors"

var (
	ErrStorageUnavailable = errors.New("file storage is unavailable")
	ErrInvalidUpload      = errors.New("invalid uploaded file")
)
