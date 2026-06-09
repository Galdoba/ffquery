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

func (m *Media) ScanRmsLevels(audio ...int) error {
	asi, err := m.collectAudioStreamInfo(audio...)
	if err != nil {
		return fmt.Errorf("failed to collect audio stream info: %w", err)
	}
	cmd, paths, err := generateLoudnessStatsScanCommand(m.Path, asi, filepath.Dir(m.Path))
	if err != nil {
		return fmt.Errorf("failed to generate ffmpeg commend: %w", err)
	}
	return m.executeAudioScanCommand(context.Background(), cmd, paths)
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
			fmt.Println(a.raw.CodecType, a.raw.Channels)
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

func generateLoudnessStatsScanCommand(inputFile string, streams []AudioStreamInfo, outputDir string) (*exec.Cmd, map[string]string, error) {
	if len(streams) == 0 {
		return nil, nil, errors.New("at least one audio stream must be provided")
	}

	var filterParts []string
	var mapArgs []string
	outputFiles := make(map[string]string)
	outNamePrefix := filepath.Base(inputFile)
	outNamePrefix = strings.TrimSuffix(outNamePrefix, filepath.Ext(outNamePrefix))

	for _, s := range streams {
		streamTag := fmt.Sprintf("stream_%d", s.Index)
		fileName := fmt.Sprintf("%s%s", outNamePrefix, streammap.NewAstatFileSuffix(s.Index))
		filePath := filepath.Join(outputDir, fileName)
		// Приводим к прямым слешам для безопасной передачи в фильтр
		slashedPath := filepath.ToSlash(filePath)
		outputFiles[fmt.Sprintf("%d", s.Index)] = slashedPath

		astat, err := filters.NewAstat(
			filters.AstatMetadata(true),
			filters.AstatReset(1),
			filters.AstatMeasurePerChannel(filters.RMSPeak, filters.PeakLevel),
			filters.AstatMeasureOverall(filters.RMSPeak, filters.PeakLevel),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create astat filter: %w", err)
		}

		filterParts = append(filterParts,
			fmt.Sprintf("[0:a:%d]asetnsamples=%d,%s,ametadata=mode=print:file='%s'[%s]",
				s.Index, s.IntervalSamples, astat.String(), slashedPath, streamTag))

		// Для exec.Cmd каждый map‑аргумент передаётся отдельно
		mapArgs = append(mapArgs, "-map", fmt.Sprintf("[%s]", streamTag))
	}

	filterComplex := strings.Join(filterParts, ";")
	progressFile := filepath.Join(outputDir, outNamePrefix+".progress")
	slashedProgress := filepath.ToSlash(progressFile)
	outputFiles[progressFileKey] = slashedProgress
	outputFiles[csvFileKey] = filepath.Join(outputDir, outNamePrefix+".AstatsScan.csv")
	// Формируем слайс аргументов для exec.Cmd
	args := []string{
		"-hide_banner", "-v", "error",
		"-progress", slashedProgress,
		"-i", inputFile,
		"-filter_complex", filterComplex,
	}
	args = append(args, mapArgs...)
	args = append(args, "-f", "null", "-")

	cmd := exec.Command("ffmpeg", args...)

	return cmd, outputFiles, nil
}

func isStatLine(line string) bool {
	if !strings.Contains(line, "lavfi.astats.") {
		return false
	}
	if !strings.Contains(line, "RMS_level") && !strings.Contains(line, "Peak_level") {
		return false
	}
	return true
}

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
	s := "scanning: ["
	for i := 10.0; i < percent; i = i + 10 {
		s += "="
	}
	s += ">"
	for len(s) < 21 {
		s += " "
	}
	s += "] "
	fmt.Printf("%s%.2f%%\r", s, percent)
}

// parseAudioStatsFile парсит файл с метаданными аудиопотока.
// Возвращает любые данные (в реальности – структуру с уровнями RMS и пиков).
func parseAudioStatsFile(filePath string) (interface{}, error) {
	// Заглушка: возвращаем имя файла, чтобы было видно, что файл обработан.
	return fmt.Sprintf("stats from %s", filePath), nil
}

// analyzeAndPrintResults принимает карту с результатами парсинга по ключам потоков и выводит сводку.
func analyzeAndPrintResults(results map[string]interface{}) {
	fmt.Println("\nРезультаты анализа аудио:")
	for key, val := range results {
		fmt.Printf("  Поток %s: %v\n", key, val)
	}
}

// ---- Основная функция ----

// executeAudioScanCommand запускает команду ffmpeg, мониторит файл прогресса,
// дожидается завершения и обрабатывает выходные файлы.
func (m *Media) executeAudioScanCommand(ctx context.Context, cmd *exec.Cmd, paths map[string]string) error {
	// Извлекаем путь к файлу прогресса
	progressPath, ok := paths[progressFileKey]
	if !ok {
		return fmt.Errorf("в карте paths отсутствует ключ 'progress'")
	}

	// Запускаем процесс
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("не удалось запустить команду: %w", err)
	}

	// Канал для получения результата Wait()
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Тикер для периодического чтения прогресса
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Основной цикл: ждём завершения команды, отмены контекста или нового тика
	for {
		select {
		case <-ctx.Done():
			// Контекст отменён – убиваем процесс
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			// Ожидаем завершения горутины с Wait, чтобы избежать зомби
			<-done
			return ctx.Err()

		case err := <-done:
			// Команда завершилась (успешно или с ошибкой)
			if err != nil {
				return fmt.Errorf("команда ffmpeg завершилась с ошибкой: %w", err)
			}
			goto processResults // not ideomatic but talerated in this case

		case <-ticker.C:
			currentTime := extractCurrentOutTime(progressPath)
			printProgressBar(currentTime / m.Duration)
		}
	}
	cmd.Process.Kill()

processResults:
	printProgressBar(100)
	fmt.Println() // перевод строки после прогресс-бара

	// Собираем пути к выходным файлам (все ключи, кроме "progress")
	statFiles := []string{}
	for key, filePath := range paths {
		if key == progressFileKey {
			continue
		}
		// Проверяем, что ключ – это номер аудиопотока (0,1,...)
		if _, err := strconv.Atoi(key); err != nil {
			continue
		}
		statFiles = append(statFiles, filePath)

		// stats, err := parseAudioStatsFile(filePath)
		// if err != nil {
		// 	return fmt.Errorf("ошибка парсинга файла %s: %w", filePath, err)
		// }
		// results[key] = stats
	}
	slices.Sort(statFiles)

	lm, err := streammap.ParseAstatFiles(statFiles)
	if err != nil {
		return fmt.Errorf("failed to parse astats files: %w", err)
	}

	f, err := os.Create(paths[csvFileKey])
	if err != nil {
		return fmt.Errorf("failed to create result file: %w", err)
	}
	defer f.Close()
	if err := lm.WriteWideCSV(f); err != nil {
		return err
	}
	time.Sleep(time.Second)
	for k, path := range paths {
		if k == csvFileKey {
			continue
		}
		f, _ := os.Create(path)
		f.Close()
		fmt.Println("delete", path, os.Remove(path))
	}
	return nil
}
