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

func TestWebMVideo(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	v := NewValidator()
	for _, scenario := range []struct {
		name, ext, duration    string
		live, audioOnly, valid bool
	}{
		{name: "standard WebM", ext: "webm", duration: "2", valid: true},
		{name: "browser style WebM without duration metadata", ext: "webm", duration: "2", live: true, valid: true},
		{name: "minimum duration without metadata", ext: "webm", duration: "1", live: true, valid: true},
		{name: "maximum duration without metadata", ext: "webm", duration: "20", live: true, valid: true},
		{name: "too short without metadata", ext: "webm", duration: "0.5", live: true},
		{name: "too long without metadata", ext: "webm", duration: "20.1", live: true},
		{name: "audio only", ext: "webm", duration: "2", audioOnly: true},
		{name: "renamed Matroska", ext: "mkv", duration: "2"},
		{name: "renamed MP4", ext: "mp4", duration: "2"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "clip."+scenario.ext)
			args := []string{"-v", "error", "-f", "lavfi", "-i", "color=c=black:s=16x16:r=10", "-t", scenario.duration, "-c:v", "libvpx"}
			if scenario.audioOnly {
				args = []string{"-v", "error", "-f", "lavfi", "-i", "anullsrc", "-t", scenario.duration, "-c:a", "libopus"}
			}
			if scenario.ext == "mp4" {
				args[len(args)-1] = "mpeg4"
			}
			if scenario.live {
				args = append(args, "-live", "1")
			}
			if output, err := exec.Command("ffmpeg", append(args, path)...).CombinedOutput(); err != nil {
				t.Fatalf("create test video: %v %s", err, output)
			}
			if scenario.live {
				output, err := exec.Command(v.ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", path).CombinedOutput()
				if err != nil || strings.TrimSpace(string(output)) != "N/A" {
					t.Fatalf("expected missing duration metadata: %v %s", err, output)
				}
			}
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			result, err := v.Video(context.Background(), ports.UploadFile{Name: "clip.WEBM", Body: file})
			if !scenario.valid {
				if err == nil {
					t.Fatal("invalid WebM accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			position, err := file.Seek(0, 1)
			if err != nil || position != 0 || result.Extension != ".webm" || result.ContentType != "video/webm" {
				t.Fatalf("unexpected WebM upload metadata: %+v, position %d, err %v", result, position, err)
			}
		})
	}
}
