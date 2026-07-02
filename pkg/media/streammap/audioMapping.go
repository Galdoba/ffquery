package streammap

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

const (
	astatsFileMarker         = ".AstatsData.Stream"
	astatsFilenameExpression = astatsFileMarker + `_(\d)\.txt$`
	astatsLineMarker         = "lavfi.astats."
	astatsLineExpression     = `^` + astatsLineMarker + `(\d+|Overall)\.(\w+)=(.+)$`
	frameExpression          = `frame:(\d+)`
	frameFactor              = 0.1
	ffmpegSilenceValue       = -750.0
)

var (
	ErrInvalidLineFormat = errors.New("invalid line format")
	ErrFrameMismatch     = errors.New("frame number mismatch")
)

type LoudnessMap struct {
	Data map[string]map[string][]float64
}

func NewLoudnessMap() *LoudnessMap {
	return &LoudnessMap{Data: make(map[string]map[string][]float64)}
}

func (lm *LoudnessMap) appendData(streamIdx, metricKey string, value float64) {
	if lm.Data[streamIdx] == nil {
		lm.Data[streamIdx] = make(map[string][]float64)
	}
	lm.Data[streamIdx][metricKey] = append(lm.Data[streamIdx][metricKey], value)
}

func NewAstatFileSuffix(streamIndex int) string {
	return fmt.Sprintf("%s_%d.txt", astatsFileMarker, streamIndex)
}

func extractStreamIndexFromPath(path string) (string, error) {
	base := filepath.Base(path)
	re := regexp.MustCompile(astatsFilenameExpression)
	match := re.FindStringSubmatch(base)
	if match == nil {
		return "", fmt.Errorf("filename %q does not match expected pattern %q", base, astatsFilenameExpression)
	}
	return match[1], nil
}

// parseStream reads ASTATS lines and populates the LoudnessMap.
// primaryKey is the stream index.
func parseStream(scanner *bufio.Scanner, primaryKey string, dataRe *regexp.Regexp, lm *LoudnessMap) error {
	var currentFrame int = -1
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, astatsLineMarker) {
			// metric line
			channel, dataType, val, err := parseAstatLine(line, dataRe)
			if err != nil {
				return err
			}
			metricKey := channel + "." + dataType
			lm.appendData(primaryKey, metricKey, val)
			continue
		}
		// frame line expected
		frame, err := extractFrameNumber(line)
		if err != nil {
			return fmt.Errorf("expected frame line, got: %q", line)
		}
		if currentFrame+1 != frame {
			return fmt.Errorf("%w: expected frame %d, got %d", ErrFrameMismatch, currentFrame+1, frame)
		}
		currentFrame = frame
	}
	return scanner.Err()
}

func parseAstatLine(line string, re *regexp.Regexp) (channel, dataType string, value float64, err error) {
	parts := re.FindStringSubmatch(line)
	if parts == nil || len(parts) != 4 {
		return "", "", 0, fmt.Errorf("%w: %q", ErrInvalidLineFormat, line)
	}
	val, err := parseValue(parts[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to parse value in %q: %w", line, err)
	}
	return parts[1], parts[2], val, nil
}

func parseValue(s string) (float64, error) {
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)
	if low == "inf" || low == "+inf" || low == "-inf" || low == "nan" {
		return ffmpegSilenceValue, nil
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return ffmpegSilenceValue, fmt.Errorf("failed to parse %q: %w", s, err)
	}
	return val, nil
}

func extractFrameNumber(line string) (int, error) {
	re := regexp.MustCompile(frameExpression)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return 0, errors.New("frame number not found")
	}
	return strconv.Atoi(matches[1])
}

// ParseAstatStreams reads ASTATS data from a set of named readers.
// Merge safely integrates data from another LoudnessMap.
// It assumes stream indices are disjoint across different calls.
func (lm *LoudnessMap) Merge(other *LoudnessMap) {
	for streamIdx, metrics := range other.Data {
		if lm.Data[streamIdx] == nil {
			lm.Data[streamIdx] = metrics
		} else {
			maps.Copy(lm.Data[streamIdx], metrics)
		}
	}
}

// ParseAstatStreams reads ASTATS data from a set of named readers concurrently.
// This avoids deadlocks when readers are backed by OS pipes (ffmpeg writes to all pipes simultaneously).
func ParseAstatStreams(streams map[string]io.Reader) (*LoudnessMap, error) {
	dataRe := regexp.MustCompile(astatsLineExpression)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		finalLM = NewLoudnessMap()
		errs    []error
	)

	for streamIdx, r := range streams {
		wg.Add(1)
		go func(idx string, reader io.Reader) {
			defer wg.Done()

			// each goroutine builds its own map to avoid contention
			lm := NewLoudnessMap()
			scanner := bufio.NewScanner(reader)
			if err := parseStream(scanner, idx, dataRe, lm); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("stream %s: %w", idx, err))
				mu.Unlock()
				return
			}

			// merge the local result into the global map under lock
			mu.Lock()
			finalLM.Merge(lm)
			mu.Unlock()
		}(streamIdx, r)
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, fmt.Errorf("parsing streams: %w", errors.Join(errs...))
	}
	return finalLM, nil
}

// WriteWideCSV writes the LoudnessMap as a wide CSV.
func (lm *LoudnessMap) WriteWideCSV(w io.Writer) error {
	if lm == nil || lm.Data == nil {
		return errors.New("LoudnessMap is nil or empty")
	}

	columns, totalFrames := lm.collectCSVColumns()
	writer := csv.NewWriter(w)

	if err := writeCSVHeader(writer, columns); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if err := writeCSVRows(writer, columns, totalFrames); err != nil {
		return err
	}

	writer.Flush()
	return writer.Error()
}

func (lm *LoudnessMap) ToCSV() (string, error) {
	var buf strings.Builder
	if err := lm.WriteWideCSV(&buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type csvColumn struct {
	header string
	values []float64
}

func (lm *LoudnessMap) collectCSVColumns() ([]csvColumn, int) {
	var columns []csvColumn
	for streamIdx, metrics := range lm.Data {
		for metricKey, vals := range metrics {
			columns = append(columns, csvColumn{
				header: fmt.Sprintf("stream_%s_%s", streamIdx, metricKey),
				values: vals,
			})
		}
	}

	slices.SortFunc(columns, func(a, b csvColumn) int {
		return strings.Compare(a.header, b.header)
	})

	totalFrames := 0
	for _, col := range columns {
		if len(col.values) > totalFrames {
			totalFrames = len(col.values)
		}
	}
	return columns, totalFrames
}

func writeCSVHeader(w *csv.Writer, columns []csvColumn) error {
	headers := []string{"frame", "time"}
	for _, col := range columns {
		headers = append(headers, col.header)
	}
	return w.Write(headers)
}

func writeCSVRows(w *csv.Writer, columns []csvColumn, totalFrames int) error {
	for frameIdx := range totalFrames {
		timeSec := float64(frameIdx) * frameFactor
		record := []string{
			strconv.Itoa(frameIdx),
			strconv.FormatFloat(timeSec, 'f', 3, 64),
		}

		for _, col := range columns {
			var val float64
			if frameIdx < len(col.values) {
				val = col.values[frameIdx]
			} else {
				val = ffmpegSilenceValue
			}
			record = append(record, strconv.FormatFloat(val, 'f', 6, 64))
		}

		if err := w.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV row for frame %d: %w", frameIdx, err)
		}
	}
	return nil
}

// CSV reading helpers (unchanged, but included for completeness)
type columnMapping struct {
	streamIdx string
	metricKey string
}

func LoudnessMapFromCSV(csvPath string) (*LoudnessMap, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %s: %w", csvPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	columns, err := parseCSVHeader(header)
	if err != nil {
		return nil, err
	}

	lm := NewLoudnessMap()
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV record: %w", err)
		}
		if err := lm.populateFromCSVRow(record, columns); err != nil {
			return nil, err
		}
	}
	return lm, nil
}

func parseCSVHeader(header []string) ([]columnMapping, error) {
	if len(header) < 2 || header[0] != "frame" || header[1] != "time" {
		return nil, fmt.Errorf("invalid CSV header: expected 'frame','time', got %v", header)
	}

	cols := make([]columnMapping, 0, len(header)-2)
	for i := 2; i < len(header); i++ {
		colName := header[i]
		parts := strings.SplitN(colName, "_", 3)
		if len(parts) != 3 || parts[0] != "stream" {
			return nil, fmt.Errorf("invalid column name %q: expected 'stream_<idx>_<metricKey>'", colName)
		}
		cols = append(cols, columnMapping{
			streamIdx: parts[1],
			metricKey: parts[2],
		})
	}
	return cols, nil
}

func (lm *LoudnessMap) populateFromCSVRow(record []string, cols []columnMapping) error {
	if len(record) < 2 {
		return fmt.Errorf("CSV record too short: %v", record)
	}

	for idx, col := range cols {
		val := ffmpegSilenceValue
		if idx+2 < len(record) {
			var err error
			val, err = parseValue(record[idx+2])
			if err != nil {
				return fmt.Errorf("failed to parse value for column %q in record %v: %w", col.metricKey, record, err)
			}
		}
		lm.appendData(col.streamIdx, col.metricKey, val)
	}
	return nil
}
