package general

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/domain/ports"
)

func parseEmployerMultipart(ctx *gin.Context, maxBytes int64) bool {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBytes)
	err := ctx.Request.ParseMultipartForm(8 * 1024 * 1024)
	if err == nil {
		return true
	}
	if ctx.Request.MultipartForm != nil {
		ctx.Request.MultipartForm.RemoveAll()
	}
	status := http.StatusBadRequest
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		status = http.StatusRequestEntityTooLarge
	}
	ctx.JSON(status, ErrorResponse{Message: "invalid or oversized multipart body"})
	return false
}

type employerMultipartFiles struct{ closers []io.Closer }

func (f *employerMultipartFiles) close() {
	for _, file := range f.closers {
		file.Close()
	}
}

func (f *employerMultipartFiles) open(ctx *gin.Context, name string, required bool) (*ports.UploadFile, error) {
	files := ctx.Request.MultipartForm.File[name]
	if len(files) == 0 && !required {
		return nil, nil
	}
	if len(files) != 1 {
		return nil, errors.New("exactly one " + name + " is required")
	}
	file, err := files[0].Open()
	if err != nil {
		return nil, err
	}
	f.closers = append(f.closers, file)
	return &ports.UploadFile{Name: files[0].Filename, Body: file}, nil
}
