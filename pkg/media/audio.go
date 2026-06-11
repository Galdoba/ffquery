package mediagroup

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
	"github.com/Galdoba/ffquery/pkg/ffprobe"
	"github.com/Galdoba/ffquery/pkg/media/streammap"
)

const (
	minimalIntervalSamples        = 200
	defaultIntervalDurationFactor = 10
	command_ScanRMS               = "RMS"
	progressFileKey               = "progress"
	csvFileKey                    = "csv"
)

func (m *Media) ScanAstats(ctx context.Context, per_channel []filters.AstatMeasure, overall []filters.AstatMeasure) error {
	cmd, paths, err := m.generateLoudnessStatsScanCommand(per_channel, overall)
	if err != nil {
		return fmt.Errorf("failed to generate ffmpeg commend: %w", err)
	}
	os.Pipe()
	return m.executeAudioScanCommand(ctx, cmd, paths)
}

func (m *Media) collectAudioStreamInfo(audio ...int) ([]AudioStreamInfo, error) {
	asi := []AudioStreamInfo{}
	for i, a := range m.Audio {
		if len(audio) != 0 && !slices.Contains(audio, i) {
			continue
		}
		as := AudioStreamInfo{
			Index:         i,
			ChannelLayout: a.raw.ChannelLayout,
			Channels:      setChannelTags(a.raw),
		}
		if len(as.Channels) < 1 {
			return nil, fmt.Errorf("unknown channel layout for audio %d of %s: %q", i, m.Path, as.ChannelLayout)
		}
		intervalSamples := a.raw.SampleRateHz() / defaultIntervalDurationFactor
		if intervalSamples < minimalIntervalSamples {
			return nil, fmt.Errorf("sample rate for audio %d of %s: is low as fuck: %dHz", i, m.Path, a.raw.SampleRateHz())
		}
		as.IntervalSamples = intervalSamples
		asi = append(asi, as)
	}
	return asi, nil
}

// ChannelNames maps known ffmpeg channel layouts to the individual channel labels.
var ChannelNames = map[string][]string{
	"mono":      {"m"},
	"stereo":    {"L", "R"},
	"5.0":       {"L", "R", "C", "LB", "RB"},
	"5.1":       {"L", "R", "C", "LFE", "LB", "RB"},
	"5.1(side)": {"L", "R", "C", "LFE", "LS", "RS"},
	"6.1":       {"L", "R", "C", "LFE", "LB", "RB", "BC"},
	"7.1":       {"L", "R", "C", "LFE", "LB", "RB", "LS", "RS"},
}

func setChannelTags(r ffprobe.Stream) []string {
	lay := r.ChannelLayout
	chans := ChannelNames[lay]
	if len(chans) == 0 {
		for i := 1; i <= r.Channels; i++ {
			chans = append(chans, fmt.Sprintf("%dch", i))
		}
	}
	return chans
}

// AudioStreamInfo holds all necessary information about one audio stream.
type AudioStreamInfo struct {
	Index           int
	ChannelLayout   string
	Channels        []string
	IntervalSamples int
}

func (m *Media) generateLoudnessStatsScanCommand(per_channel []filters.AstatMeasure, overall []filters.AstatMeasure) (*exec.Cmd, map[string]string, error) {
	streams, err := m.collectAudioStreamInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to collect audio stream info: %w", err)
	}
	if len(streams) == 0 {
		return nil, nil, errors.New("at least one audio stream must be provided")
	}
	var filterParts []string
	var mapArgs []string
	dir := filepath.Dir(m.Path)
	outputFiles := make(map[string]string)
	outNamePrefix := filepath.Base(m.Path)
	outNamePrefix = strings.TrimSuffix(outNamePrefix, filepath.Ext(outNamePrefix))

	for _, s := range streams {
		streamTag := fmt.Sprintf("stream_%d", s.Index)
		fileName := fmt.Sprintf("%s%s", outNamePrefix, streammap.NewAstatFileSuffix(s.Index))
		filePath := filepath.Join(dir, fileName)
		slashedPath := filepath.ToSlash(filePath)
		outputFiles[fmt.Sprintf("%d", s.Index)] = slashedPath

		astat, err := filters.NewAstat(
			filters.AstatMetadata(true),
			filters.AstatReset(1),
			filters.AstatMeasurePerChannel(per_channel...),
			filters.AstatMeasureOverall(overall...),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create astat filter: %w", err)
		}

		filterParts = append(filterParts,
			fmt.Sprintf("[0:a:%d]asetnsamples=%d,%s,ametadata=mode=print:file='%s'[%s]",
				s.Index, s.IntervalSamples, astat.String(), slashedPath, streamTag))

		mapArgs = append(mapArgs, "-map", fmt.Sprintf("[%s]", streamTag))
	}

	filterComplex := strings.Join(filterParts, ";")
	progressFile := filepath.Join(dir, outNamePrefix+".progress")
	slashedProgress := filepath.ToSlash(progressFile)
	outputFiles[progressFileKey] = slashedProgress
	outputFiles[csvFileKey] = filepath.Join(dir, outNamePrefix+".AstatsScan.csv")

	args := []string{
		"-hide_banner", "-v", "error",
		"-progress", slashedProgress,
		"-i", m.Path,
		"-filter_complex", filterComplex,
	}
	args = append(args, mapArgs...)
	args = append(args, "-f", "null", "-")

	cmd := exec.Command("ffmpeg", args...)
	return cmd, outputFiles, nil
}

// func isStatLine(line string) bool {
// 	if !strings.Contains(line, "lavfi.astats.") {
// 		return false
// 	}
// 	if !strings.Contains(line, "RMS_level") && !strings.Contains(line, "Peak_level") {
// 		return false
// 	}
// 	return true
// }

func extractCurrentOutTime(path string) float64 {
	file, err := os.Open(path)
	if err != nil {
		return -1
	}
	defer file.Close()
	const prefix = "out_time_us=" //ffmpeg prints as nanoseconds
	var lastOutTimeUs int64 = -1
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		valStr := line[len(prefix):]
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err == nil {
			lastOutTimeUs = val
		}
	}

	if err := scanner.Err(); err != nil {
		return -1
	}
	if lastOutTimeUs < 0 {
		return -1
	}

	return float64(lastOutTimeUs) / 1000000.0 * 100.0
}

func printProgressBar(percent float64) {
	percent = max(percent, 0)
	percent = min(percent, 100)
	if percent == 100 {
		fmt.Fprint(os.Stderr, "scanning: [==========] 100.00%\r")
	}
	s := "scanning: ["
	for i := 10.0; i < percent; i = i + 10 {
		s += "="
	}
	s += ">"
	for len(s) < 21 {
		s += " "
	}
	s += "] "
	fmt.Fprintf(os.Stderr, "%s%.2f%%\r", s, percent)
}

// ---- Основная функция ----

// // executeAudioScanCommand запускает команду ffmpeg, мониторит файл прогресса,
// // дожидается завершения и обрабатывает выходные файлы.
// func (m *Media) executeAudioScanCommand(ctx context.Context, cmd *exec.Cmd, paths map[string]string) error {
// 	progressPath, ok := paths[progressFileKey]
// 	if !ok {
// 		return fmt.Errorf("paths does not contain 'progress'")
// 	}

// 	fmt.Fprintf(os.Stderr, "run command: %v\n", cmd.Args)
// 	if err := cmd.Start(); err != nil {
// 		return fmt.Errorf("failed to run command: %w", err)
// 	}

// 	done := make(chan error, 1)
// 	go func() {
// 		done <- cmd.Wait()
// 	}()

// 	progressReadTicker := time.NewTicker(500 * time.Millisecond)
// 	defer progressReadTicker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			if cmd.Process != nil {
// 				_ = cmd.Process.Kill()
// 			}
// 			<-done
// 			return ctx.Err()

// 		case err := <-done:
// 			if err != nil {
// 				return fmt.Errorf("command stopped with error: %w", err)
// 			}
// 			goto processResults // not ideomatic but talerated in this case

// 		case <-progressReadTicker.C:
// 			currentTime := extractCurrentOutTime(progressPath)
// 			printProgressBar(currentTime / m.Duration)
// 		}
// 	}

// processResults:
// 	printProgressBar(100)
// 	fmt.Println() // перевод строки после прогресс-бара

// 	// Собираем пути к выходным файлам (все ключи, кроме "progress")
// 	statFiles := []string{}
// 	for key, filePath := range paths {
// 		if key == progressFileKey {
// 			continue
// 		}
// 		// Проверяем, что ключ – это номер аудиопотока (0,1,...)
// 		if _, err := strconv.Atoi(key); err != nil {
// 			continue
// 		}
// 		statFiles = append(statFiles, filePath)

// 	}
// 	slices.Sort(statFiles)

// 	lm, err := streammap.ParseAstatFiles(statFiles)
// 	if err != nil {
// 		return fmt.Errorf("failed to parse astats files: %w", err)
// 	}

// 	f, err := os.Create(paths[csvFileKey])
// 	if err != nil {
// 		return fmt.Errorf("failed to create result file: %w", err)
// 	}
// 	defer f.Close()
// 	if err := lm.WriteWideCSV(f); err != nil {
// 		return err
// 	}
// 	time.Sleep(time.Second)
// 	for k, path := range paths {
// 		if k == csvFileKey {
// 			continue
// 		}
// 		f, _ := os.Create(path)
// 		f.Close()
// 		fmt.Println("delete", path, os.Remove(path))
// 	}
// 	return nil
// }

func (m *Media) executeAudioScanCommand(ctx context.Context, cmd *exec.Cmd, paths map[string]string) error {
	progressPath, ok := paths[progressFileKey]
	if !ok {
		return fmt.Errorf("paths does not contain 'progress'")
	}

	fmt.Fprintf(os.Stderr, "run command: %v\n", fmt.Sprintf(strings.Join(cmd.Args, " ")))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	// Wait for command completion in a separate goroutine
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Monitor progress and command completion
	err := m.monitorCommandProgress(ctx, cmd, done, progressPath)
	if err != nil {
		return err
	}

	// Command finished successfully – process results
	printProgressBar(100)
	fmt.Println() // newline after the progress bar

	// Parse astats files and write CSV
	if err := m.writeWideCSVFromStats(paths); err != nil {
		return err
	}

	// Clean up temporary files (all except the CSV result)
	m.cleanupTempFiles(paths)

	return nil
}

// monitorCommandProgress watches the command's execution, kills it on context cancellation,
// and updates the progress bar based on the progress file.
func (m *Media) monitorCommandProgress(ctx context.Context, cmd *exec.Cmd, done <-chan error, progressPath string) error {
	progressTicker := time.NewTicker(500 * time.Millisecond)
	defer progressTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-done
			return ctx.Err()

		case err := <-done:
			// Command finished
			if err != nil {
				return fmt.Errorf("command stopped with error: %w", err)
			}
			return nil // success

		case <-progressTicker.C:
			currentTime := extractCurrentOutTime(progressPath)
			printProgressBar(currentTime / m.Duration)
		}
	}
}

// writeWideCSVFromStats collects all astats files (keys that are numeric stream indices),
// parses them, and writes the wide CSV result.
func (m *Media) writeWideCSVFromStats(paths map[string]string) error {
	statFiles := m.collectStatFiles(paths)

	lm, err := streammap.ParseAstatFiles(statFiles)
	if err != nil {
		return fmt.Errorf("failed to parse astats files: %w", err)
	}

	csvPath, ok := paths[csvFileKey]
	if !ok {
		return fmt.Errorf("paths does not contain CSV file key")
	}

	f, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("failed to create result file: %w", err)
	}
	defer f.Close()

	if err := lm.WriteWideCSV(f); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	if m.Meta == nil {
		m.Meta = make(map[string]string)
	}
	m.Meta["astatsCSV"] = csvPath

	return nil
}

// collectStatFiles returns a sorted slice of file paths whose keys are numeric strings
// (representing audio stream indices), skipping the progress and CSV files.
func (m *Media) collectStatFiles(paths map[string]string) []string {
	var statFiles []string
	for key, filePath := range paths {
		if key == progressFileKey || key == csvFileKey {
			continue
		}
		// Only accept keys that are valid integers (stream numbers)
		if _, err := strconv.Atoi(key); err != nil {
			continue
		}
		statFiles = append(statFiles, filePath)
	}
	slices.Sort(statFiles)
	return statFiles
}

// cleanupTempFiles deletes all temporary files except the final CSV.
// Errors are logged to stderr but do not stop the function.
func (m *Media) cleanupTempFiles(paths map[string]string) {
	for key, path := range paths {
		if key == csvFileKey {
			continue
		}
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(os.Stderr, "failed to remove %s: %v\n", path, err)
		} else {
			fmt.Println("delete", path)
		}
	}
}
