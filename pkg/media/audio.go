package mediagroup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
	"github.com/Galdoba/ffquery/pkg/ffprobe"
)

const (
	minimalIntervalSamples        = 200
	defaultIntervalDurationFactor = 10
	command_ScanRMS               = "RMS"
)

func (m *Media) ScanRmsLevels(audio ...int) error {
	// строим команду
	asi, err := m.collectAudioStreamInfo(audio...)
	if err != nil {
		return fmt.Errorf("failed to collect audio stream info: %w", err)
	}
	cmd, paths, err := generateLoudnessStatsScanCommand(m.Path, asi, filepath.Dir(m.Path))
	if err != nil {
		return fmt.Errorf("failed to generate ffmpeg commend: %w", err)
	}
	return err
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

// func generateLoudnessStatsScanCommand(inputFile string, streams []AudioStreamInfo, outputDir string) (string, map[string]string, error) {
// 	if len(streams) == 0 {
// 		return "", nil, errors.New("at least one audio stream must be provided")
// 	}

// 	var filterParts []string
// 	var mapParts []string
// 	outputFiles := make(map[string]string)
// 	outNamePrefix := filepath.Base(inputFile)
// 	outNamePrefix = strings.TrimSuffix(outNamePrefix, filepath.Ext(outNamePrefix))

// 	for _, s := range streams {
// 		streamTag := fmt.Sprintf("stream_%d", s.Index)
// 		fileName := fmt.Sprintf("%s_stream_%d.txt", outNamePrefix, s.Index)
// 		filePath := filepath.Join(outputDir, fileName)
// 		outputFiles[fmt.Sprintf("%d", s.Index)] = filePath

// 		astat, err := filters.NewAstat(
// 			filters.AstatMetadata(true),
// 			filters.AstatReset(1),
// 			filters.AstatMeasurePerChannel(filters.RMSPeak, filters.PeakLevel),
// 			filters.AstatMeasureOverall(filters.RMSPeak, filters.PeakLevel),
// 		)
// 		if err != nil {
// 			return "", nil, fmt.Errorf("failed to create astat filter: %w", err)
// 		}

// 		filterParts = append(filterParts,
// 			fmt.Sprintf("[0:a:%d]asetnsamples=%d,%s,ametadata=mode=print:file='%s'[%s]",
// 				s.Index, s.IntervalSamples, astat.String(), filePath, streamTag))
// 		mapParts = append(mapParts, fmt.Sprintf("-map [%s]", streamTag))
// 	}

// 	filterComplex := strings.Join(filterParts, ";")
// 	mapArgs := strings.Join(mapParts, " ")
// 	progressFile := filepath.Join(outputDir, outNamePrefix+".progress")

// 	cmd := fmt.Sprintf("ffmpeg -hide_banner -v error -progress %s -i %s -filter_complex \"%s\" %s -f null -",
// 		progressFile,
// 		inputFile, filterComplex, mapArgs)
// 	outputFiles["progress"] = progressFile

// 	cmd = strings.ReplaceAll(cmd, "\\", "/")
// 	for k := range outputFiles {
// 		outputFiles[k] = filepath.ToSlash(outputFiles[k])
// 	}

// 	return cmd, outputFiles, nil
// }

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
		fileName := fmt.Sprintf("%s_stream_%d.txt", outNamePrefix, s.Index)
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
	outputFiles["progress"] = slashedProgress

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

// RunCommandString разбирает строку команды и выполняет её через exec.Command.
// Поддерживает одинарные и двойные кавычки, пробелы внутри кавычек не разбивают аргумент.
// Экранирование символов обратным слешем не реализовано (в ваших данных не требуется).
func RunCommandString(cmdLine string) error {
	args, err := parseCommandLine(cmdLine)
	if err != nil {
		return fmt.Errorf("ошибка парсинга команды: %w", err)
	}
	if len(args) == 0 {
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// parseCommandLine разбивает командную строку на аргументы, поддерживая кавычки.
func parseCommandLine(s string) ([]string, error) {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)
	i := 0
	n := len(s)

	for i < n {
		ch := s[i]

		// Если не в кавычках, ищем пробельные символы для разделения аргументов
		if !inQuote && unicode.IsSpace(rune(ch)) {
			// Пробелы – разделители аргументов
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			i++
			continue
		}

		// Обработка открытия/закрытия кавычек
		if !inQuote && (ch == '"' || ch == '\'') {
			inQuote = true
			quoteChar = ch
			i++
			continue
		}

		if inQuote && ch == quoteChar {
			inQuote = false
			quoteChar = 0
			i++
			continue
		}

		// Обычный символ – добавляем в текущий аргумент
		current.WriteByte(ch)
		i++
	}

	// Последний аргумент
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if inQuote {
		return nil, fmt.Errorf("незакрытая кавычка %c", quoteChar)
	}

	return args, nil
}
