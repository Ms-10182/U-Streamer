package ffmpeg

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

// VideoInfo holds the probed video metadata
type VideoInfo struct {
	Width  int
	Height int
}

// probeResult maps the ffprobe JSON output
type probeResult struct {
	Streams []probeStream `json:"streams"`
}

type probeStream struct {
	CodecType string `json:"codec_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// Probe uses ffprobe to detect the resolution of a video file
func Probe(filePath string) (*VideoInfo, error) {
	log.Printf("[FFPROBE] Probing file: %s", filePath)

	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "v:0",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result probeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			info := &VideoInfo{
				Width:  stream.Width,
				Height: stream.Height,
			}
			log.Printf("[FFPROBE] Detected resolution: %dx%d", info.Width, info.Height)
			return info, nil
		}
	}

	return nil, fmt.Errorf("no video stream found in %s", filePath)
}
