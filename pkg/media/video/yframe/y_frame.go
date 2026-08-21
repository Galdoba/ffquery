// File: scanner.go
package yframe

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
	"github.com/Galdoba/ffquery/pkg/media/video/blocksize"
	"github.com/Galdoba/ffquery/pkg/media/video/metricfile"
	"github.com/Galdoba/ffquery/pkg/progress"
)

// ffmpeg argument constants.
const (
	ffmpegCommand    = "ffmpeg"
	ffmpegInputFlag  = "-i"
	ffmpegFilterFlag = "-vf"
	ffmpegFormatGray = "format=gray"
	ffmpegFormatFlag = "-f"
	ffmpegRawVideo   = "rawvideo"
	ffmpegPixFmtFlag = "-pix_fmt"
	ffmpegGrayFmt    = "gray"
	ffmpegPipeOut    = "pipe:1"
)

// metadataProvider abstracts obtaining video stream metadata.
type metadataProvider func(videoPath string) (*ffprobe.Stream, error)

// ffmpegStarter abstracts starting an ffmpeg process that outputs raw gray frames.
// It returns a reader for the frames, a cleanup function (to wait for the process),
// and an error.
type ffmpegStarter func(videoPath string) (io.ReadCloser, func() error, error)

// Scanner orchestrates the Y‑block metric scan using injected dependencies.
// It automatically reports progress via an internal tracker writing to os.Stderr.
type Scanner struct {
	getMeta metadataProvider
	startFF ffmpegStarter
	tracker *progress.Tracker
}

// NewScanner creates a Scanner with injected metadata and ffmpeg providers.
// It also initializes a default progress tracker with output to stderr.
// Use the returned scanner immediately; the tracker is started lazily on Scan.
func NewScanner(getMeta metadataProvider, startFF ffmpegStarter) (*Scanner, error) {
	if getMeta == nil {
		return nil, fmt.Errorf("metadata provider must not be nil")
	}
	if startFF == nil {
		return nil, fmt.Errorf("ffmpeg starter must not be nil")
	}
	// Create tracker with sensible defaults. Total steps will be set once metadata is available.
	tracker := progress.NewTracker(
		progress.WithDescription("Scanning video"),
		progress.WithSpinner(progress.CommonSpinnerFrames()),
		progress.WithOutput(os.Stderr),
		progress.WithAutoStart(false), // will be started explicitly in Scan
	)

	return &Scanner{
		getMeta: getMeta,
		startFF: startFF,
		tracker: tracker,
	}, nil
} // ScanToFile is a convenience wrapper that builds a real Scanner and runs Scan.
// It uses the actual ffprobe and ffmpeg executables.
func ScanToFile(videoPath string, cfg blocksize.CalculationConfig, outputPath string) error {
	realMeta := func(path string) (*ffprobe.Stream, error) {
		raw, err := ffprobe.NewRawData(path)
		if err != nil {
			return nil, fmt.Errorf("ffprobe: %w", err)
		}
		stream, err := raw.GetVideo()
		if err != nil {
			return nil, fmt.Errorf("no video stream: %w", err)
		}
		return stream, nil
	}

	realFF := func(path string) (io.ReadCloser, func() error, error) {
		cmd := exec.Command(ffmpegCommand,
			ffmpegInputFlag, path,
			ffmpegFilterFlag, ffmpegFormatGray,
			ffmpegFormatFlag, ffmpegRawVideo,
			ffmpegPixFmtFlag, ffmpegGrayFmt,
			ffmpegPipeOut,
		)
		cmd.Stderr = nil
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, nil, fmt.Errorf("stdout pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return nil, nil, fmt.Errorf("start ffmpeg: %w", err)
		}
		cleanup := func() error {
			return cmd.Wait()
		}
		return stdout, cleanup, nil
	}

	scanner, err := NewScanner(realMeta, realFF)
	if err != nil {
		return fmt.Errorf("create scanner: %w", err)
	}
	return scanner.Scan(videoPath, cfg, outputPath)
}

// Scan runs the full Y‑block metric generation and reports progress.
func (s *Scanner) Scan(videoPath string, cfg blocksize.CalculationConfig, outputPath string) error {
	// 1. Metadata
	stream, err := s.getMeta(videoPath)
	if err != nil {
		return fmt.Errorf("metadata: %w", err)
	}
	width, height := stream.Width, stream.Height
	fpsNum, fpsDen := stream.FPSNumDen()
	if fpsNum == 0 || fpsDen == 0 {
		return fmt.Errorf("invalid frame rate in video stream: %s", stream.RFrameRate)
	}

	// Estimate total frames for the progress bar.
	totalFrames := estimateTotalFrames(stream)
	s.tracker.SetTotal(totalFrames)

	// 2. Block grid
	grid, err := blocksize.CalculateBlockSizes(cfg, width, height)
	if err != nil {
		return fmt.Errorf("blocksize: %w", err)
	}

	// 3. Output file
	writer, closeWriter, err := openMetricWriter(outputPath, grid, width, height, fpsNum, fpsDen)
	if err != nil {
		return fmt.Errorf("open metric writer: %w", err)
	}
	defer closeWriter()

	// 4. Frame source
	frameSource, cleanup, err := s.startFF(videoPath)
	if err != nil {
		return fmt.Errorf("start frame source: %w", err)
	}
	defer cleanup()

	// 5. Start progress tracking
	s.tracker.Start()
	defer s.tracker.Close()

	// 6. Process frames with progress updates
	if err := processFrames(frameSource, writer, grid, width, s.tracker); err != nil {
		return fmt.Errorf("process frames: %w", err)
	}
	return nil
}

// estimateTotalFrames tries to derive the number of frames from video stream metadata.
func estimateTotalFrames(stream *ffprobe.Stream) int64 {
	// Priority: NbFrames field, otherwise duration × fps.
	if stream.Frames() > 0 {
		return int64(stream.Frames())
	}
	fpsNum, fpsDen := stream.FPSNumDen()
	if fpsNum > 0 && fpsDen > 0 {
		durationSec := stream.DurationSeconds()
		if durationSec > 0 {
			fps := float64(fpsNum) / float64(fpsDen)
			return int64(durationSec * fps)
		}
	}
	// If unknown, tracker will show indeterminate progress.
	return 0
}

// openMetricWriter creates the output file, writes the metric header,
// and returns the writer along with a function that must be called to close the file.
func openMetricWriter(path string, grid blocksize.Grid, width, height, fpsNum, fpsDen int) (*metricfile.Writer, func() error, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("create output file: %w", err)
	}
	closeFile := func() error {
		return f.Close()
	}

	hdr := metricfile.NewHeader(
		grid.HorizontalBlocks,
		grid.VerticalBlocks,
		width,
		height,
		fpsNum,
		fpsDen,
	)
	writer, err := metricfile.NewWriter(f, hdr)
	if err != nil {
		f.Close() // close file on error to avoid leak
		return nil, nil, fmt.Errorf("write header: %w", err)
	}
	return writer, closeFile, nil
}

// processFrames теперь принимает трекер.
func processFrames(src io.Reader, writer *metricfile.Writer, grid blocksize.Grid, width int, tracker *progress.Tracker) error {
	frameBuf := make([]byte, grid.FrameYBufferSize())
	for {
		_, err := io.ReadFull(src, frameBuf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read frame: %w", err)
		}
		averages := computeBlockAveragesParallel(frameBuf, width, grid)
		if err := writer.WriteFrame(averages); err != nil {
			return fmt.Errorf("write frame: %w", err)
		}
		// Update progress after each successfully written frame.
		if tracker != nil {
			_ = tracker.Increment()
		}
	}
}
