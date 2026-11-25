package media

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

// ffprobe groups WebM and Matroska together. The EBML header's DocType
// distinguishes WebM from a Matroska file renamed to .webm.
func hasWebMDocType(header []byte) bool {
	magic := []byte{0x1a, 0x45, 0xdf, 0xa3}
	if !bytes.HasPrefix(header, magic) {
		return false
	}
	header = header[len(magic):]
	size, width := ebmlSize(header)
	if width == 0 || size > uint64(len(header)-width) {
		return false
	}
	elements := header[width : width+int(size)]
	for len(elements) != 0 {
		idWidth, _ := ebmlWidth(elements[0])
		if idWidth == 0 || idWidth > 4 || len(elements) < idWidth {
			return false
		}
		id := elements[:idWidth]
		elements = elements[idWidth:]
		size, width = ebmlSize(elements)
		if width == 0 || size > uint64(len(elements)-width) {
			return false
		}
		end := width + int(size)
		if bytes.Equal(id, []byte{0x42, 0x82}) {
			return string(elements[width:end]) == "webm"
		}
		elements = elements[end:]
	}
	return false
}

func ebmlWidth(first byte) (int, byte) {
	for width, mask := 1, byte(0x80); mask != 0; width, mask = width+1, mask>>1 {
		if first&mask != 0 {
			return width, mask
		}
	}
	return 0, 0
}

func ebmlSize(data []byte) (uint64, int) {
	if len(data) == 0 {
		return 0, 0
	}
	width, mask := ebmlWidth(data[0])
	if width == 0 || len(data) < width {
		return 0, 0
	}
	size := uint64(data[0] & (mask - 1))
	for _, next := range data[1:width] {
		size = size<<8 | uint64(next)
	}
	if size == (uint64(1)<<(7*width))-1 {
		return 0, 0 // Unknown sizes are invalid in the EBML header.
	}
	return size, width
}

func (v *Validator) webmPacketDuration(ctx context.Context, path string) (float64, error) {
	command := exec.CommandContext(ctx, v.ffprobe, "-v", "error", "-protocol_whitelist", "file",
		"-show_entries", "packet=pts_time,duration_time", "-of", "compact=p=0:nk=0", path)
	output, err := command.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := command.Start(); err != nil {
		return 0, err
	}
	start, end := math.Inf(1), math.Inf(-1)
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		var pts = math.NaN()
		var duration float64
		for _, field := range strings.Split(scanner.Text(), "|") {
			key, value, _ := strings.Cut(field, "=")
			number, err := strconv.ParseFloat(value, 64)
			if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
				continue
			}
			switch key {
			case "pts_time":
				pts = number
			case "duration_time":
				duration = max(0, number)
			}
		}
		if !math.IsNaN(pts) {
			start = min(start, pts)
			end = max(end, pts+duration)
		}
	}
	scanErr := scanner.Err()
	if scanErr != nil {
		output.Close()
	}
	waitErr := command.Wait()
	if scanErr != nil {
		return 0, scanErr
	}
	if waitErr != nil {
		return 0, waitErr
	}
	if math.IsInf(start, 0) || math.IsInf(end, 0) {
		return 0, fmt.Errorf("WebM packet timestamps are missing")
	}
	// ffprobe prints timestamps as decimal seconds. Round subtraction to the same precision.
	return math.Round((end-start)*1e6) / 1e6, nil
}
