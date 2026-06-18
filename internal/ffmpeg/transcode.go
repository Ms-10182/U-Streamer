package ffmpeg

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Rendition defines a target resolution for transcoding
type Rendition struct {
	Name         string // e.g. "360p"
	Width        int
	Height       int
	VideoBitrate string // e.g. "800k"
	AudioBitrate string // e.g. "64k"
	MaxRate      string
	BufSize      string
}

// Available renditions ordered from lowest to highest
var AllRenditions = []Rendition{
	{Name: "360p", Width: 640, Height: 360, VideoBitrate: "800k", AudioBitrate: "64k", MaxRate: "856k", BufSize: "1200k"},
	{Name: "480p", Width: 854, Height: 480, VideoBitrate: "1400k", AudioBitrate: "128k", MaxRate: "1498k", BufSize: "2100k"},
	{Name: "720p", Width: 1280, Height: 720, VideoBitrate: "2800k", AudioBitrate: "128k", MaxRate: "2996k", BufSize: "4200k"},
	{Name: "1080p", Width: 1920, Height: 1080, VideoBitrate: "5000k", AudioBitrate: "192k", MaxRate: "5350k", BufSize: "7500k"},
}

// SelectRenditions picks renditions that are <= the source resolution
// Never upscales — only creates lower or equal resolution versions
func SelectRenditions(sourceWidth, sourceHeight int) []Rendition {
	var selected []Rendition
	for _, r := range AllRenditions {
		if r.Height <= sourceHeight {
			selected = append(selected, r)
		}
	}
	// If source is smaller than 360p, still create a 360p-bitrate version at source res
	if len(selected) == 0 {
		selected = append(selected, Rendition{
			Name:         "native",
			Width:        sourceWidth,
			Height:       sourceHeight,
			VideoBitrate: "600k",
			AudioBitrate: "64k",
			MaxRate:      "642k",
			BufSize:      "900k",
		})
	}
	return selected
}

// Transcode takes a source video and creates multi-resolution HLS output
func Transcode(sourcePath, outputDir string, renditions []Rendition) error {
	log.Printf("[FFMPEG] Transcoding %s into %d renditions", sourcePath, len(renditions))

	// Create output directories
	for _, r := range renditions {
		dir := filepath.Join(outputDir, r.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output dir %s: %w", dir, err)
		}
	}

	args := buildFFmpegArgs(sourcePath, outputDir, renditions)

	log.Printf("[FFMPEG] Running: ffmpeg %s", strings.Join(args, " "))

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg transcoding failed: %w", err)
	}

	// Generate master playlist manually for better control
	if err := generateMasterPlaylist(outputDir, renditions); err != nil {
		return fmt.Errorf("failed to generate master playlist: %w", err)
	}

	log.Printf("[FFMPEG] Transcoding complete for %s", sourcePath)
	return nil
}

// buildFFmpegArgs constructs the ffmpeg command arguments
func buildFFmpegArgs(sourcePath, outputDir string, renditions []Rendition) []string {
	args := []string{
		"-i", sourcePath,
		"-y", // overwrite output
	}

	// Map input streams for each rendition
	for range renditions {
		args = append(args, "-map", "0:v:0", "-map", "0:a:0?")
	}

	// Set codec defaults
	args = append(args, "-c:v", "libx264", "-c:a", "aac", "-ar", "48000")

	// Per-rendition settings
	for i, r := range renditions {
		vi := fmt.Sprintf("v:%d", i)
		ai := fmt.Sprintf("a:%d", i)

		args = append(args,
			fmt.Sprintf("-filter:v:%d", i), fmt.Sprintf("scale=w=%d:h=%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", r.Width, r.Height, r.Width, r.Height),
			fmt.Sprintf("-b:%s", vi), r.VideoBitrate,
			fmt.Sprintf("-maxrate:%s", vi), r.MaxRate,
			fmt.Sprintf("-bufsize:%s", vi), r.BufSize,
			fmt.Sprintf("-b:%s", ai), r.AudioBitrate,
		)
	}

	// Keyframe alignment — critical for ABR switching
	args = append(args,
		"-g", "48",
		"-keyint_min", "48",
		"-sc_threshold", "0",
		"-preset", "fast",
	)

	// Build var_stream_map
	var streamMap []string
	for i, r := range renditions {
		streamMap = append(streamMap, fmt.Sprintf("v:%d,a:%d,name:%s", i, i, r.Name))
	}
	args = append(args, "-var_stream_map", strings.Join(streamMap, " "))

	// HLS output settings
	args = append(args,
		"-hls_time", "4",
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_list_size", "0",
		"-hls_segment_filename", filepath.Join(outputDir, "%v", "segment%03d.ts"),
		"-f", "hls",
	)

	// Output playlist pattern
	args = append(args, filepath.Join(outputDir, "%v", "index.m3u8"))

	return args
}

// generateMasterPlaylist creates the master.m3u8 file
func generateMasterPlaylist(outputDir string, renditions []Rendition) error {
	var lines []string
	lines = append(lines, "#EXTM3U")
	lines = append(lines, "#EXT-X-VERSION:3")
	lines = append(lines, "")

	for _, r := range renditions {
		bandwidth := estimateBandwidth(r.VideoBitrate, r.AudioBitrate)
		lines = append(lines,
			fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,NAME=\"%s\"",
				bandwidth, r.Width, r.Height, r.Name),
			fmt.Sprintf("%s/index.m3u8", r.Name),
			"",
		)
	}

	content := strings.Join(lines, "\n")
	masterPath := filepath.Join(outputDir, "master.m3u8")

	log.Printf("[FFMPEG] Writing master playlist: %s", masterPath)
	return os.WriteFile(masterPath, []byte(content), 0644)
}

// estimateBandwidth converts bitrate strings like "800k" to bits per second
func estimateBandwidth(videoBitrate, audioBitrate string) int {
	vbr := parseBitrate(videoBitrate)
	abr := parseBitrate(audioBitrate)
	return vbr + abr
}

func parseBitrate(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "k") {
		s = strings.TrimSuffix(s, "k")
		val := 0
		fmt.Sscanf(s, "%d", &val)
		return val * 1000
	}
	if strings.HasSuffix(s, "m") {
		s = strings.TrimSuffix(s, "m")
		val := 0
		fmt.Sscanf(s, "%d", &val)
		return val * 1000000
	}
	val := 0
	fmt.Sscanf(s, "%d", &val)
	return val
}
