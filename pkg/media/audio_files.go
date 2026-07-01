package mediagroup

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
	"github.com/Galdoba/ffquery/pkg/media/streammap"
)

// generateFileCommand builds an ffmpeg command that writes astats metadata to files.
func (scan *scanConfig) generateFileCommand() error {
	var filterParts []string
	var mapArgs []string
	outputPaths := make(map[string]string)

	for _, s := range scan.Streams {
		streamKey := fmt.Sprintf("%d", s.Index)
		fileName := scan.Prefix + streammap.NewAstatFileSuffix(s.Index)
		filePath := filepath.Join(scan.Dir, fileName)
		slashed := filepath.ToSlash(filePath)
		outputPaths[streamKey] = slashed

		astat, err := filters.NewAstat(
			filters.AstatMetadata(true),
			filters.AstatReset(1),
			filters.AstatMeasurePerChannel(scan.PerChannel...),
			filters.AstatMeasureOverall(scan.Overall...),
		)
		if err != nil {
			return fmt.Errorf("creating astat filter for stream %d: %w", s.Index, err)
		}

		streamTag := fmt.Sprintf("stream_%d", s.Index)
		filterParts = append(filterParts,
			fmt.Sprintf("[0:a:%d]asetnsamples=%d,%s,ametadata=mode=4:file='%s'[%s]",
				s.Index, s.IntervalSamples, astat.String(), slashed, streamTag))
		mapArgs = append(mapArgs, "-map", fmt.Sprintf("[%s]", streamTag))
	}

	progressPath := filepath.Join(scan.Dir, scan.Prefix+".progress")
	slashedProgress := filepath.ToSlash(progressPath)
	outputPaths[progressFileKey] = slashedProgress

	args := []string{
		"-hide_banner", "-v", "error",
		"-progress", slashedProgress,
		"-i", scan.InputPath,
		"-filter_complex", strings.Join(filterParts, ";"),
	}
	args = append(args, mapArgs...)
	args = append(args, "-f", "null", "-")

	scan.cmd = exec.Command("ffmpeg", args...)
	scan.output = &audioScanOutputs{
		Mode:    "files",
		Paths:   outputPaths,
		CSVPath: scan.CSVPath,
	}
	return nil
}

// executeFileScan runs the ffmpeg command that writes progress and astats to files.
func (m *Media) executeFileScan(ctx context.Context, scan *scanConfig) error {
	progressPath, ok := scan.output.Paths[progressFileKey]
	if !ok {
		return errors.New("progress file path not found")
	}

	var stderrBuf bytes.Buffer
	scan.cmd.Stderr = &stderrBuf
	scan.cmd.Stdout = &stderrBuf
	fmt.Fprintf(os.Stderr, "run command: \n%v\n", strings.Join(scan.cmd.Args, " "))

	if scan.progressTracker != nil {
		scan.progressTracker.Start()
	}

	if err := scan.cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	done := make(chan error, 1)
	go func() { done <- scan.cmd.Wait() }()

	if err := m.monitorProgressFromFile(ctx, scan, done, progressPath); err != nil {
		fmt.Fprintf(os.Stderr, "=== ffmpeg output ===\n%s\n===================\n", stderrBuf.String())
		return err
	}

	if scan.progressTracker != nil {
		scan.progressTracker.Done()
		scan.progressTracker.Close()
	}
	lm, err := m.parseAstatFiles(scan.output.Paths)
	if err != nil {
		return fmt.Errorf("parsing astats files: %w", err)
	}
	if err := m.writeWideCSVFromLoudnessMap(lm, scan.output.CSVPath); err != nil {
		return err
	}
	m.cleanupTempFiles(scan.output.Paths, scan.output.CSVPath)
	return nil
}

// parseAstatFiles opens the temporary files and feeds them to ParseAstatStreams.
func (m *Media) parseAstatFiles(paths map[string]string) (*streammap.LoudnessMap, error) {
	readers := make(map[string]io.Reader)
	var closers []io.Closer
	for key, path := range paths {
		if key == progressFileKey || key == csvFileKey {
			continue
		}
		if _, err := strconv.Atoi(key); err != nil {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			for _, c := range closers {
				c.Close()
			}
			return nil, fmt.Errorf("opening %s: %w", path, err)
		}
		readers[key] = f
		closers = append(closers, f)
	}
	defer func() {
		for _, c := range closers {
			c.Close()
		}
	}()
	// fmt.Println("go to streammap")
	return streammap.ParseAstatStreams(readers)
}

// monitorProgressFromFile polls the progress file and updates the bar until the command completes.
func (m *Media) monitorProgressFromFile(ctx context.Context, scan *scanConfig, done <-chan error, progressPath string) error {
	ticker := time.NewTicker(progressTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if scan.cmd.Process != nil {
				_ = scan.cmd.Process.Kill()
			}
			<-done
			return ctx.Err()
		case err := <-done:
			if err != nil {
				return fmt.Errorf("ffmpeg error: %w", err)
			}
			return nil
		case <-ticker.C:
			us, err := readLastOutTimeUs(progressPath)
			if err == nil && us >= 0 {
				pct := usToPercent(us, m.Duration)
				if scan.progressTracker != nil {
					scan.progressTracker.SetPct(pct)
				}
			}
		}
	}
}

// readLastOutTimeUs reads the progress file and returns the last out_time_us value in microseconds.
func readLastOutTimeUs(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return -1, err
	}
	defer f.Close()

	var lastUs int64 = -1
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		us, err := parseOutTimeUs(scanner.Text())
		if err == nil {
			lastUs = us
		}
	}
	if err := scanner.Err(); err != nil {
		return -1, err
	}
	return lastUs, nil
}

// cleanupTempFiles removes temporary astats files, keeping the final CSV.
func (m *Media) cleanupTempFiles(paths map[string]string, csvPath string) {
	for _, path := range paths {
		if path == csvPath {
			continue
		}
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(os.Stderr, "failed to remove %s: %v\n", path, err)
		}
	}
}
