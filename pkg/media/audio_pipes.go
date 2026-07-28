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
	"strings"
	"sync"
	"time"

	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
	"github.com/Galdoba/ffquery/pkg/media/streammap"
	"github.com/Galdoba/ffquery/pkg/progress"
)

// generatePipeCommand builds an ffmpeg command that communicates via pipes,
// producing both total and speech astats data for each audio stream.
func (scanCmd *scanConfig) generatePipeCommand() error {
	out := &audioScanOutputs{
		Mode:        "pipes",
		PipeReaders: make(map[string]io.ReadCloser),
		CSVPath:     scanCmd.CSVPath,
	}

	var writers []io.Closer
	cmd := exec.Command("ffmpeg", "-hide_banner", "-v", "error")

	// Progress pipe (fd 3)
	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("progress pipe: %w", err)
	}
	out.PipeReaders[progressFileKey] = pr
	writers = append(writers, pw)
	cmd.Args = append(cmd.Args, "-progress", "pipe:3")
	cmd.ExtraFiles = append(cmd.ExtraFiles, pw)

	var filterParts []string
	var mapArgs []string

	for i, s := range scanCmd.Streams {
		// Allocate two file descriptors per stream: total and speech
		fdTotal := 4 + 2*i
		fdSpeech := 5 + 2*i

		streamIdx := fmt.Sprintf("%d", s.Index)

		// Create pipes for total and speech
		prTotal, pwTotal, err := os.Pipe()
		if err != nil {
			return fmt.Errorf("stream %d total pipe: %w", s.Index, err)
		}
		prSpeech, pwSpeech, err := os.Pipe()
		if err != nil {
			return fmt.Errorf("stream %d speech pipe: %w", s.Index, err)
		}

		out.PipeReaders[streamIdx+"_total"] = prTotal
		out.PipeReaders[streamIdx+"_speech"] = prSpeech

		writers = append(writers, pwTotal, pwSpeech)
		cmd.ExtraFiles = append(cmd.ExtraFiles, pwTotal, pwSpeech)

		astat, err := filters.NewAstat(
			filters.AstatMetadata(true),
			filters.AstatReset(1),
			filters.AstatMeasurePerChannel(scanCmd.PerChannel...),
			filters.AstatMeasureOverall(scanCmd.Overall...),
		)
		if err != nil {
			return fmt.Errorf("creating astat filter for stream %d: %w", s.Index, err)
		}

		tagAll := fmt.Sprintf("all_%d", s.Index)
		tagSpeech := fmt.Sprintf("speech_%d", s.Index)
		tagTotalOut := fmt.Sprintf("total_out_%d", s.Index)
		tagSpeechOut := fmt.Sprintf("speech_out_%d", s.Index)

		// asplit into two chains, then total and bandpass+speech
		filterParts = append(filterParts,
			fmt.Sprintf("[0:a:%d]asplit=2[%s][%s]", s.Index, tagAll, tagSpeech),
			fmt.Sprintf("[%s]asetnsamples=%d,%s,ametadata=mode=4:file='pipe\\:%d'[%s]",
				tagAll, s.IntervalSamples, astat.String(), fdTotal, tagTotalOut),
			fmt.Sprintf("[%s]bandpass=f=300:w=3400,asetnsamples=%d,%s,ametadata=mode=4:file='pipe\\:%d'[%s]",
				tagSpeech, s.IntervalSamples, astat.String(), fdSpeech, tagSpeechOut),
		)

		mapArgs = append(mapArgs,
			"-map", fmt.Sprintf("[%s]", tagTotalOut),
			"-map", fmt.Sprintf("[%s]", tagSpeechOut),
		)
	}

	cmd.Args = append(cmd.Args, "-i", scanCmd.InputPath,
		"-filter_complex", joinFilterComplex(filterParts...))
	cmd.Args = append(cmd.Args, mapArgs...)
	cmd.Args = append(cmd.Args, "-f", "null", "-")

	out.closeWriters = func() {
		for _, w := range writers {
			w.Close()
		}
	}
	scanCmd.cmd = cmd
	scanCmd.output = out
	return nil
}

func joinFilterComplex(fcParts ...string) string {
	return strings.Join(fcParts, ";")
}

// executePipeScan runs the ffmpeg command, monitors progress and parses astats from pipes.
func (m *Media) executePipeScan(ctx context.Context, scan *scanConfig) error {
	var stderrBuf bytes.Buffer
	scan.cmd.Stderr = &stderrBuf
	scan.cmd.Stdout = &stderrBuf
	fmt.Fprintf(os.Stderr, "run command: %v\n", strings.Join(scan.cmd.Args, " "))

	if scan.progressTracker != nil {
		scan.progressTracker.Start()
	}

	if err := scan.cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}
	scan.output.closeWriters()

	parsedCh := parseAstatPipes(scan.output.PipeReaders)
	done := make(chan error, 1)
	go func() { done <- scan.cmd.Wait() }()

	progressReader, ok := scan.output.PipeReaders[progressFileKey]
	if !ok {
		return errors.New("progress pipe reader missing")
	}
	if err := trackProgressFromPipe(ctx, progressReader, m.Duration, done, scan.progressTracker); err != nil {
		fmt.Fprintf(os.Stderr, "=== ffmpeg output ===\n%s\n===================\n", stderrBuf.String())
		return err
	}

	if scan.progressTracker != nil {
		scan.progressTracker.Done()
		scan.progressTracker.Close()
	}

	res := <-parsedCh
	if res.err != nil {
		return fmt.Errorf("astats parsing: %w", res.err)
	}
	return m.writeWideCSVFromLoudnessMap(res.lm, scan.output.CSVPath)
}

// parseAstatPipes extracts non-progress pipe readers and feeds them to ParseAstatStreams in a goroutine.
func parseAstatPipes(readers map[string]io.ReadCloser) chan struct {
	lm  *streammap.LoudnessMap
	err error
} {
	ch := make(chan struct {
		lm  *streammap.LoudnessMap
		err error
	}, 1)
	go func() {
		pipeMap := make(map[string]io.Reader)
		for k, r := range readers {
			if k == progressFileKey {
				continue
			}
			// All remaining keys (e.g., "0_total", "0_speech") are astats pipes
			pipeMap[k] = r
		}
		lm, err := streammap.ParseAstatStreams(pipeMap)
		ch <- struct {
			lm  *streammap.LoudnessMap
			err error
		}{lm, err}
	}()
	return ch
}

// trackProgressFromPipe updates the progress tracker (if not nil) and returns an error if the context is cancelled or ffmpeg fails.
func trackProgressFromPipe(ctx context.Context, r io.Reader, totalDuration float64, done <-chan error, tracker *progress.Tracker) error {
	var (
		percent float64
		mu      sync.Mutex
	)
	stopProgress := make(chan struct{})
	defer close(stopProgress)
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			us, err := parseOutTimeUs(scanner.Text())
			if err == nil {
				pct := usToPercent(us, totalDuration)
				mu.Lock()
				percent = pct
				if tracker != nil {
					tracker.SetPct(pct)
				}
				mu.Unlock()
			}
		}
	}()

	ticker := time.NewTicker(progressTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			if err != nil {
				return fmt.Errorf("ffmpeg error: %w", err)
			}
			// On successful completion ensure 100% is shown
			mu.Lock()
			if percent < 100 && tracker != nil {
				tracker.SetPct(100)
			}
			mu.Unlock()
			return nil
		case <-ticker.C:
			// ticker is used only to check context/completion,
			// progress updates happen in the goroutine
		}
	}
}
