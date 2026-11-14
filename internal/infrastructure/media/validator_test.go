package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"zedlancer/backend/internal/domain/ports"
)

func TestDocumentContentAndLimits(t *testing.T) {
	v := NewValidator()
	ctx := context.Background()
	var picture bytes.Buffer
	canvas := image.NewRGBA(image.Rect(0, 0, 2, 2))
	canvas.Set(0, 0, color.White)
	if err := png.Encode(&picture, canvas); err != nil {
		t.Fatal(err)
	}
	for _, file := range []ports.UploadFile{{Name: "id.pdf", Body: strings.NewReader("%PDF-1.7\ncontent")}, {Name: "id.png", Body: bytes.NewReader(picture.Bytes())}} {
		result, err := v.Document(ctx, file)
		if err != nil {
			t.Fatal(err)
		}
		position, _ := file.Body.Seek(0, 1)
		if position != 0 || result.Size == 0 {
			t.Fatal("validated file was not rewound")
		}
	}
	for _, file := range []ports.UploadFile{{Name: "id.pdf", Body: strings.NewReader("not a PDF")}, {Name: "id.jpg", Body: bytes.NewReader(picture.Bytes())}, {Name: "id.exe", Body: strings.NewReader("%PDF-")}, {Name: "id.pdf", Body: strings.NewReader("")}, {Name: "id.pdf", Body: strings.NewReader(strings.Repeat("a", int(MaxDocumentBytes)+1))}} {
		if _, err := v.Document(ctx, file); err == nil {
			t.Fatalf("invalid document accepted: %s", file.Name)
		}
	}
}

func TestVideoDurationAndActualContainer(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	v := NewValidator()
	for _, duration := range []string{"1", "2", "20", "0.5", "20.1", "30", "60"} {
		path := filepath.Join(t.TempDir(), "verification.mp4")
		command := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=black:s=16x16:r=10", "-t", duration, "-c:v", "mpeg4", "-y", path)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("create test video: %v %s", err, output)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		_, validationErr := v.Video(context.Background(), ports.UploadFile{Name: "verification.mp4", Body: file})
		file.Close()
		if (validationErr == nil) != (duration == "1" || duration == "2" || duration == "20") {
			t.Fatalf("duration %s: %v", duration, validationErr)
		}
	}
	if _, err := v.Video(context.Background(), ports.UploadFile{Name: "fake.mp4", Body: strings.NewReader("not a video")}); err == nil {
		t.Fatal("fake video accepted")
	}
}
