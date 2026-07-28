package yframe

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
	"github.com/Galdoba/ffquery/pkg/media/video/blocksize"
	"github.com/Galdoba/ffquery/pkg/media/video/metricfile"
)

// ScanToFile scans the video file, calculates block averages according to cfg,
// and writes the resulting Y-block metric file to outputPath.
func ScanToFile(videoPath string, cfg blocksize.CalculationConfig, outputPath string) error {
	// 1. Get video metadata
	raw, err := ffprobe.NewRawData(videoPath)
	if err != nil {
		return fmt.Errorf("ffprobe: %w", err)
	}
	videoStream, err := raw.GetVideo()
	if err != nil {
		return fmt.Errorf("no video stream: %w", err)
	}
	width, height := videoStream.Width, videoStream.Height
	fpsNum, fpsDen := videoStream.FPSNumDen()
	if fpsNum == 0 || fpsDen == 0 {
		return fmt.Errorf("invalid frame rate in video stream: %s", videoStream.RFrameRate)
	}

	// 2. Calculate block grid
	grid, err := blocksize.CalculateBlockSizes(cfg, width, height)
	if err != nil {
		return fmt.Errorf("blocksize: %w", err)
	}

	// 3. Open output file and write header
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

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
		return fmt.Errorf("write header: %w", err)
	}

	// 4. Start ffmpeg
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vf", "format=gray",
		"-f", "rawvideo",
		"-pix_fmt", "gray",
		"pipe:1",
	)
	cmd.Stderr = nil
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}
	defer cmd.Wait()

	// 5. Process frames
	frameBuf := make([]byte, grid.FrameYBufferSize())
	for {
		_, err := io.ReadFull(stdout, frameBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read frame: %w", err)
		}
		vector := computeBlockAverages(frameBuf, width, grid)
		if err := writer.WriteFrame(vector); err != nil {
			return fmt.Errorf("write frame: %w", err)
		}
	}
	return nil
}

// computeBlockAverages calculates the average Y value for each block in the grid.
// `frame` is a full Y-plane with row-major layout of size width × height.
// The grid is guaranteed to tile the frame perfectly (width % BlockWidth == 0, etc.).
func computeBlockAverages(frame []byte, width int, grid blocksize.Grid) []byte {
	numBlocks := grid.HorizontalBlocks * grid.VerticalBlocks
	vector := make([]byte, numBlocks)
	idx := 0
	for row := 0; row < grid.VerticalBlocks; row++ {
		for col := 0; col < grid.HorizontalBlocks; col++ {
			// Calculate pixel offset of the top-left corner of this block.
			startY := row * grid.BlockHeight
			startX := col * grid.BlockWidth

			var sum int
			for y := 0; y < grid.BlockHeight; y++ {
				offset := (startY+y)*width + startX
				for x := 0; x < grid.BlockWidth; x++ {
					sum += int(frame[offset+x])
				}
			}
			avg := sum / (grid.BlockWidth * grid.BlockHeight)
			vector[idx] = byte(avg)
			idx++
		}
	}
	return vector
}
