package ports

import (
	"context"
	"io"
)

type UploadFile struct {
	Name string
	Body io.ReadSeeker
}

type ValidatedFile struct {
	UploadFile
	ContentType string
	Size        int64
	Extension   string
}

type MediaValidator interface {
	Document(context.Context, UploadFile) (*ValidatedFile, error)
	Video(context.Context, UploadFile) (*ValidatedFile, error)
}
