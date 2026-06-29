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
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
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

// usePipes определяет, использовать ли пайпы (не Windows) или файлы (Windows)
var usePipes = runtime.GOOS != "windows"

// audioScanOutputs хранит информацию о целевых выводах ffmpeg.
type audioScanOutputs struct {
	Mode         string                   // "files" или "pipes"
	Paths        map[string]string        // для файлового режима
	PipeReaders  map[string]io.ReadCloser // для пайпов: ключи "progress", "0","1"...
	closeWriters func()                   // закрыть писателей пайпов после Start
	CSVPath      string
	closeWriter  func()
}

func (m *Media) ScanAstats(ctx context.Context, per_channel []filters.AstatMeasure, overall []filters.AstatMeasure) error {
	cmd, outputs, err := m.generateLoudnessStatsScanCommand(per_channel, overall)
	if err != nil {
		return fmt.Errorf("failed to generate ffmpeg command: %w", err)
	}
	fmt.Println("cmd:", cmd.Args)

	return m.executeAudioScanCommand(ctx, cmd, outputs)
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

func (m *Media) generateLoudnessStatsScanCommand(per_channel []filters.AstatMeasure, overall []filters.AstatMeasure) (*exec.Cmd, *audioScanOutputs, error) {
	streams, err := m.collectAudioStreamInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to collect audio stream info: %w", err)
	}
	if len(streams) == 0 {
		return nil, nil, errors.New("at least one audio stream must be provided")
	}

	dir := filepath.Dir(m.Path)
	outNamePrefix := filepath.Base(m.Path)
	outNamePrefix = strings.TrimSuffix(outNamePrefix, filepath.Ext(outNamePrefix))
	csvPath := filepath.Join(dir, outNamePrefix+".AstatsScan.csv")

	if usePipes {
		return m.generatePipeCommand(streams, per_channel, overall, dir, outNamePrefix, csvPath)
	}
	return m.generateFileCommand(streams, per_channel, overall, dir, outNamePrefix, csvPath)
}

func (m *Media) generateFileCommand(streams []AudioStreamInfo, per_channel, overall []filters.AstatMeasure, dir, prefix, csvPath string) (*exec.Cmd, *audioScanOutputs, error) {
	var filterParts []string
	var mapArgs []string
	outputFiles := make(map[string]string)

	for _, s := range streams {
		streamTag := fmt.Sprintf("stream_%d", s.Index)
		fileName := fmt.Sprintf("%s%s", prefix, streammap.NewAstatFileSuffix(s.Index))
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
			fmt.Sprintf("[0:a:%d]asetnsamples=%d,%s,ametadata=mode=4:file='%s'[%s]",
				s.Index, s.IntervalSamples, astat.String(), slashedPath, streamTag))
		mapArgs = append(mapArgs, "-map", fmt.Sprintf("[%s]", streamTag))
	}

	filterComplex := strings.Join(filterParts, ";")
	progressPath := filepath.Join(dir, prefix+".progress")
	slashedProgress := filepath.ToSlash(progressPath)
	outputFiles[progressFileKey] = slashedProgress

	args := []string{
		"-hide_banner", "-v", "error",
		"-progress", slashedProgress,
		"-i", m.Path,
		"-filter_complex", filterComplex,
	}
	args = append(args, mapArgs...)
	args = append(args, "-f", "null", "-")

	cmd := exec.Command("ffmpeg", args...)
	return cmd, &audioScanOutputs{
		Mode:    "files",
		Paths:   outputFiles,
		CSVPath: csvPath,
	}, nil
}

// generatePipeCommand – пайповая версия для Linux/других ОС.
func (m *Media) generatePipeCommand(streams []AudioStreamInfo, per_channel, overall []filters.AstatMeasure, dir, prefix, csvPath string) (*exec.Cmd, *audioScanOutputs, error) {
	out := &audioScanOutputs{
		Mode:        "pipes",
		PipeReaders: make(map[string]io.ReadCloser),
		CSVPath:     csvPath,
	}

	var writers []io.Closer
	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-v", "error",
		// -progress будет добавлен ниже после создания пайпа
	)

	// Прогресс-пайп (fd 3)
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, nil, fmt.Errorf("progress pipe: %w", err)
	}
	out.PipeReaders[progressFileKey] = pr
	writers = append(writers, pw)
	cmd.Args = append(cmd.Args, "-progress", "pipe:3")
	cmd.ExtraFiles = append(cmd.ExtraFiles, pw)

	// Пайпы для каждого аудиопотока (fd 4+)
	var filterParts []string
	var mapArgs []string
	for i, s := range streams {
		fd := 4 + i // progress занял fd 3
		streamTag := fmt.Sprintf("stream_%d", s.Index)

		pr, pw, err := os.Pipe()
		if err != nil {
			return nil, nil, fmt.Errorf("stream %d pipe: %w", s.Index, err)
		}
		out.PipeReaders[fmt.Sprintf("%d", s.Index)] = pr
		writers = append(writers, pw)
		cmd.ExtraFiles = append(cmd.ExtraFiles, pw)

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
			fmt.Sprintf("[0:a:%d]asetnsamples=%d,%s,ametadata=mode=4:file='pipe\\:%d'[%s]",
				s.Index, s.IntervalSamples, astat.String(), fd, streamTag))
		mapArgs = append(mapArgs, "-map", fmt.Sprintf("[%s]", streamTag))
	}

	cmd.Args = append(cmd.Args, "-i", m.Path, "-filter_complex", strings.Join(filterParts, ";"))
	cmd.Args = append(cmd.Args, mapArgs...)
	cmd.Args = append(cmd.Args, "-f", "null", "-")

	out.closeWriters = func() {
		for _, w := range writers {
			fmt.Println("close writer", w)
			w.Close()
		}
	}
	return cmd, out, nil
}

// ---- Основная функция ----

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

// executeAudioScanCommand теперь работает с audioScanOutputs.
func (m *Media) executeAudioScanCommand(ctx context.Context, cmd *exec.Cmd, outputs *audioScanOutputs) error {
	if outputs.Mode == "pipes" {
		return m.executePipeScan(ctx, cmd, outputs)
	}
	return m.executeFileScan(ctx, cmd, outputs)
}

// executeFileScan – оригинальное выполнение для файлов (Windows).
func (m *Media) executeFileScan(ctx context.Context, cmd *exec.Cmd, outputs *audioScanOutputs) error {
	progressPath, ok := outputs.Paths[progressFileKey]
	if !ok {
		return fmt.Errorf("paths does not contain 'progress'")
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	cmd.Stdout = &stderrBuf
	fmt.Fprintf(os.Stderr, "run command: %v\n", strings.Join(cmd.Args, " "))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	if err := m.monitorCommandProgressFromFile(ctx, cmd, done, progressPath); err != nil {
		fmt.Fprintf(os.Stderr, "=== ffmpeg output ===\n%s\n===================\n", stderrBuf.String())
		return err
	}

	printProgressBar(100)
	fmt.Println()

	if err := m.writeWideCSVFromPaths(outputs.Paths, outputs.CSVPath); err != nil {
		return err
	}
	m.cleanupTempFiles(outputs.Paths, outputs.CSVPath)
	return nil
}

// executePipeScan – выполнение с пайпами (Linux).
func (m *Media) executePipeScan(ctx context.Context, cmd *exec.Cmd, outputs *audioScanOutputs) error {
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	cmd.Stdout = &stderrBuf
	fmt.Fprintf(os.Stderr, "run command: %v\n", strings.Join(cmd.Args, " "))

	fmt.Println(outputs)
	fmt.Println("===")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	// Закрываем писателей в родительском процессе, чтобы дочерний мог получить EOF при завершении
	// outputs.closeWriters()

	// Горутина для парсинга astats – она использует существующий ParseAstatReaders
	type parsedResult struct {
		lm  *streammap.LoudnessMap
		err error
	}
	parsedCh := make(chan parsedResult, 1)
	go func() {
		readers := make(map[int]io.Reader)
		for k, r := range outputs.PipeReaders {
			if k == progressFileKey {
				continue
			}
			idx, _ := strconv.Atoi(k)
			readers[idx] = r
		}
		lm, err := streammap.ParseAstatReaders(readers)
		parsedCh <- parsedResult{lm, err}
	}()

	// Мониторинг прогресса из пайпа
	progressReader := outputs.PipeReaders[progressFileKey]
	if progressReader == nil {
		return errors.New("progress pipe reader missing")
	}
	var progressPercent float64
	var progressMu sync.Mutex
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// Горутина обновления прогресса
	_, cancelProgress := context.WithCancel(context.Background())
	defer cancelProgress()
	go func() {
		defer cancelProgress()
		updateProgressFromPipe(progressReader, &progressPercent, &progressMu)
	}()

	// Тикер для отображения прогресса
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Основной цикл ожидания
	for {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-done
			return ctx.Err()

		case err := <-done:
			// Команда завершилась
			cancelProgress() // остановить обновление прогресса
			outputs.closeWriters()
			if err != nil {
				fmt.Fprintf(os.Stderr, "=== ffmpeg output ===\n%s\n===================\n", stderrBuf.String())
				return fmt.Errorf("command stopped with error: %w", err)
			}
			// Дожидаемся завершения парсинга
			res := <-parsedCh
			if res.err != nil {
				return fmt.Errorf("astats parsing failed: %w", res.err)
			}
			// Пишем итоговый CSV
			if err := m.writeWideCSVFromLoudnessMap(res.lm, outputs.CSVPath); err != nil {
				return err
			}
			printProgressBar(100)
			fmt.Println()
			return nil

		case <-ticker.C:
			progressMu.Lock()
			pct := progressPercent
			progressMu.Unlock()
			printProgressBar(pct)
		}
	}
}

// updateProgressFromPipe читает строки из пайпа прогресса и обновляет процент.
func updateProgressFromPipe(r io.Reader, pct *float64, mu *sync.Mutex) {
	fmt.Println(r, pct)
	scanner := bufio.NewScanner(r)
	const prefix = "out_time_us="
	var lastUs int64 = -1
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		valStr := line[len(prefix):]
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err == nil {
			lastUs = val
			mu.Lock()
			*pct = float64(lastUs) / 1000000.0 * 100.0
			mu.Unlock()
		}
	}
	// после EOF не трогаем процент (может быть не 100, если ffmpeg не дописал)
}

// writeWideCSVFromPaths – оригинальный парсинг файлов (Windows).
func (m *Media) writeWideCSVFromPaths(paths map[string]string, csvPath string) error {
	statFiles := m.collectStatFiles(paths)
	lm, err := streammap.ParseAstatFiles(statFiles)
	if err != nil {
		return fmt.Errorf("failed to parse astats files: %w", err)
	}
	return m.writeWideCSVFromLoudnessMap(lm, csvPath)
}

// writeWideCSVFromLoudnessMap записывает итоговый CSV из LoudnessMap.
func (m *Media) writeWideCSVFromLoudnessMap(lm *streammap.LoudnessMap, csvPath string) error {
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

// collectStatFiles – остаётся без изменений.
func (m *Media) collectStatFiles(paths map[string]string) []string {
	var statFiles []string
	for key, filePath := range paths {
		if key == progressFileKey || key == csvFileKey {
			continue
		}
		if _, err := strconv.Atoi(key); err != nil {
			continue
		}
		statFiles = append(statFiles, filePath)
	}
	slices.Sort(statFiles)
	return statFiles
}

// cleanupTempFiles – удаляет временные файлы, кроме CSV. Используется только в файловом режиме.
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

// --- мониторинг прогресса для файлов (оставлен для Windows) ---
func (m *Media) monitorCommandProgressFromFile(ctx context.Context, cmd *exec.Cmd, done <-chan error, progressPath string) error {
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
			if err != nil {
				return fmt.Errorf("command stopped with error: %w", err)
			}
			return nil
		case <-progressTicker.C:
			currentTime := extractCurrentOutTime(progressPath)
			printProgressBar(currentTime / m.Duration)
		}
	}
}

func extractCurrentOutTime(path string) float64 {
	file, err := os.Open(path)
	if err != nil {
		return -1
	}
	defer file.Close()
	const prefix = "out_time_us="
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
