package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

const (
	MaxDocumentBytes int64 = 5 * 1024 * 1024
	MaxVideoBytes    int64 = 50 * 1024 * 1024
)

type Validator struct{ ffprobe string }

func NewValidator() *Validator {
	path := os.Getenv("FFPROBE_PATH")
	if path == "" {
		path = "ffprobe"
	}
	return &Validator{ffprobe: path}
}

func invalid(reason string) error { return fmt.Errorf("%w: %s", domainerrors.ErrInvalidUpload, reason) }

func inspect(file ports.UploadFile, max int64) (int64, []byte, error) {
	if file.Body == nil {
		return 0, nil, invalid("required file is missing")
	}
	if _, err := file.Body.Seek(0, io.SeekStart); err != nil {
		return 0, nil, invalid("cannot read file")
	}
	size, err := io.Copy(io.Discard, io.LimitReader(file.Body, max+1))
	if err != nil {
		return 0, nil, invalid("cannot read file")
	}
	if size == 0 || size > max {
		return 0, nil, invalid("file is empty or exceeds its size limit")
	}
	if _, err := file.Body.Seek(0, io.SeekStart); err != nil {
		return 0, nil, err
	}
	header := make([]byte, 512)
	n, err := file.Body.Read(header)
	if err != nil && err != io.EOF {
		return 0, nil, invalid("cannot read file")
	}
	if _, err := file.Body.Seek(0, io.SeekStart); err != nil {
		return 0, nil, err
	}
	return size, header[:n], nil
}

func (v *Validator) Document(ctx context.Context, file ports.UploadFile) (*ports.ValidatedFile, error) {
	size, header, err := inspect(file, MaxDocumentBytes)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(file.Name))
	contentType := http.DetectContentType(header)
	switch ext {
	case ".pdf":
		if !bytes.HasPrefix(header, []byte("%PDF-")) {
			return nil, invalid("document must be PDF, PNG, or JPEG")
		}
		contentType = "application/pdf"
	case ".jpg", ".jpeg", ".png":
		cfg, format, err := image.DecodeConfig(file.Body)
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 100000000 {
			return nil, invalid("invalid image")
		}
		if (ext == ".png" && format != "png") || (ext != ".png" && format != "jpeg") {
			return nil, invalid("image extension does not match content")
		}
		contentType = "image/" + format
	default:
		return nil, invalid("document must be PDF, PNG, or JPEG")
	}
	if _, err := file.Body.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return &ports.ValidatedFile{UploadFile: file, ContentType: contentType, Size: size, Extension: ext}, nil
}

func (v *Validator) Video(ctx context.Context, file ports.UploadFile) (*ports.ValidatedFile, error) {
	size, _, err := inspect(file, MaxVideoBytes)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(file.Name))
	if ext != ".mp4" && ext != ".mov" {
		return nil, invalid("video must be MP4 or MOV")
	}
	temp, err := os.CreateTemp("", "employer-video-*"+ext)
	if err != nil {
		return nil, err
	}
	defer os.Remove(temp.Name())
	if _, err := io.Copy(temp, file.Body); err != nil {
		temp.Close()
		return nil, err
	}
	if err := temp.Close(); err != nil {
		return nil, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	command := exec.CommandContext(probeCtx, v.ffprobe, "-v", "error", "-protocol_whitelist", "file", "-show_entries", "format=duration,format_name:stream=codec_type", "-of", "json", temp.Name())
	var output bytes.Buffer
	command.Stdout = &output
	if err := command.Run(); err != nil {
		if _, missing := err.(*exec.Error); missing {
			return nil, fmt.Errorf("ffprobe is required for video validation")
		}
		return nil, invalid("video cannot be inspected")
	}
	var result struct {
		Format struct {
			Duration string `json:"duration"`
			Name     string `json:"format_name"`
		} `json:"format"`
		Streams []struct {
			Type string `json:"codec_type"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		return nil, invalid("invalid video metadata")
	}
	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	hasVideo := false
	for _, stream := range result.Streams {
		if stream.Type == "video" {
			hasVideo = true
		}
	}
	if err != nil || math.IsNaN(duration) || math.IsInf(duration, 0) || duration < 1 || duration > 20 || !hasVideo || !strings.Contains(result.Format.Name, "mov") {
		return nil, invalid("video must be MP4/MOV with a video stream and a duration of 1-20 seconds")
	}
	if _, err := file.Body.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	contentType := "video/mp4"
	if ext == ".mov" {
		contentType = "video/quicktime"
	}
	return &ports.ValidatedFile{UploadFile: file, ContentType: contentType, Size: size, Extension: ext}, nil
}
