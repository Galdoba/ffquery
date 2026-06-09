package streammap

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	astatsFileMarker         = ".AstatsData.Stream"
	astatsFilenameExpression = astatsFileMarker + `_(\d)\.txt$`
	astatsLineMarker         = "lavfi.astats."
	astatsLineExpression     = `^` + astatsLineMarker + `(\d+|Overall)\.(\w+)=(.+)$`
	frameExpression          = `frame:(\d+)`
	frameFactor              = 0.1
	silenceValue             = -120.0
)

type ErrInvalidLineFormat error

func errInvalidLineFormat(line string) error {
	return fmt.Errorf("invalid line format: %q", line)
}

type LoudnessMap struct {
	Data map[string]map[string][]float64
}

func (lm *LoudnessMap) appendData(primeKey, secondaryKey string, value float64) {
	if lm.Data[primeKey] == nil {
		lm.Data[primeKey] = make(map[string][]float64)
	}
	lm.Data[primeKey][secondaryKey] = append(lm.Data[primeKey][secondaryKey], value)
}

// NewLoudnessMap создаёт пустую структуру.
func NewLoudnessMap() *LoudnessMap {
	return &LoudnessMap{
		Data: make(map[string]map[string][]float64),
	}
}

func NewAstatFileSuffix(streamIndex int) string {
	return fmt.Sprintf("%s_%d.txt", astatsFileMarker, streamIndex)
}

// ParseASTATFiles принимает список путей к файлам и заполняет LoudnessMap.
func ParseAstatFiles(filePaths []string) (*LoudnessMap, error) {
	lm := NewLoudnessMap()

	fileRe := regexp.MustCompile(astatsFilenameExpression)
	dataRe := regexp.MustCompile(astatsLineExpression)

	for _, path := range filePaths {
		fmt.Fprintf(os.Stderr, "processing %s...", path)

		scanner, err := newFileProcessor(path, fileRe, dataRe)
		if err != nil {
			return nil, fmt.Errorf("failed to create scanner with context: %w", err)
		}
		if err := scanner.processAndFill(lm); err != nil {
			return nil, fmt.Errorf("failed to process %s: %w", path, err)
		}

		fmt.Fprintf(os.Stderr, "  ok\n")
	}
	return lm, nil
}

type astatsProcessor struct {
	scanner    *bufio.Scanner
	reader     *os.File
	dataRe     *regexp.Regexp
	primaryKey string
	filepath   string
}

func newFileProcessor(path string, fileRe, dataRe *regexp.Regexp) (*astatsProcessor, error) {
	base := filepath.Base(path)
	match := fileRe.FindStringSubmatch(base)
	if match == nil {
		return nil, fmt.Errorf("failed to extract stream index: filename does not match template (*%s): %s", astatsFilenameExpression, base)
	}
	primaryKey := match[1]

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}

	scanner := bufio.NewScanner(f)
	sc := astatsProcessor{
		scanner:    scanner,
		reader:     f,
		dataRe:     dataRe,
		primaryKey: primaryKey,
		filepath:   path,
	}
	return &sc, nil
}

func (sc *astatsProcessor) processAndFill(lm *LoudnessMap) error {
	frameControlCounter := 0
	for sc.scanner.Scan() {
		line := sc.scanner.Text()

		if !strings.HasPrefix(line, astatsLineMarker) {
			if err := assertFrameNumber(frameControlCounter, line); err != nil {
				return err
			}
			frameControlCounter++
			continue
		}

		parts := sc.dataRe.FindStringSubmatch(line)
		if parts == nil || len(parts) != 4 {
			return errInvalidLineFormat(line)
		}
		channel := parts[1]  // "1", "2", "Overall"
		dataType := parts[2] // "Peak_level", "RMS_peak", ...
		rawVal := parts[3]

		val, err := parseValue(rawVal)
		if err != nil {
			return fmt.Errorf("failed to parse value in %s: %q: %w", sc.filepath, line, err)
		}

		internalKey := channel + "." + dataType
		lm.appendData(sc.primaryKey, internalKey, val)
	}

	if err := sc.scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file %s: %w", sc.filepath, err)
	}
	sc.reader.Close()

	return nil
}

var frameRe = regexp.MustCompile(frameExpression)

func extractFrameNumber(line string) (int, error) {
	matches := frameRe.FindStringSubmatch(line)
	if len(matches) < 2 {
		return 0, fmt.Errorf("номер кадра не найден в строке: %q", line)
	}
	return strconv.Atoi(matches[1])
}

func assertFrameNumber(expectedFrame int, line string) error {
	frame, err := extractFrameNumber(line)
	if err != nil {
		return fmt.Errorf("unknown line format: %q", line)
	}
	if frame != expectedFrame {
		return fmt.Errorf("unexpected frame number (%d): expect %d", frame, expectedFrame)
	}
	return nil
}

func parseValue(s string) (float64, error) {
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)
	if low == "inf" || low == "+inf" || low == "-inf" || low == "nan" {
		return silenceValue, nil
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return silenceValue, fmt.Errorf("failed to parse %q: %w", s, err)
	}
	return val, nil
}

// csvColumn describes a single output column: its header and the data slice.
type csvColumn struct {
	header string
	values []float64
}

// WriteWideCSV writes the LoudnessMap data as a wide CSV.
func (lm *LoudnessMap) WriteWideCSV(w io.Writer) error {
	if lm == nil || lm.Data == nil {
		return fmt.Errorf("LoudnessMap is nil or empty")
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
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to flush csv: %w", err)
	}
	return nil
}

// collectCSVColumns extracts all columns from the LoudnessMap,
// sorts them by header, and returns the sorted slice together with
// the maximum number of frames (the length of the longest data slice).
func (lm *LoudnessMap) collectCSVColumns() ([]csvColumn, int) {
	var columns []csvColumn
	for streamIdx, metricsMap := range lm.Data {
		for metricKey, vals := range metricsMap {
			columns = append(columns, csvColumn{
				header: fmt.Sprintf("stream_%s_%s", streamIdx, metricKey),
				values: vals,
			})
		}
	}

	slices.SortFunc(columns, func(a, b csvColumn) int {
		return strings.Compare(a.header, b.header)
	})

	var totalFrames int
	for _, col := range columns {
		if len(col.values) > totalFrames {
			totalFrames = len(col.values)
		}
	}
	return columns, totalFrames
}

// writeCSVHeader writes the header row to the CSV writer.
func writeCSVHeader(w *csv.Writer, columns []csvColumn) error {
	headers := []string{"frame", "time"}
	for _, col := range columns {
		headers = append(headers, col.header)
	}
	return w.Write(headers)
}

// writeCSVRows writes all data rows for the given columns and frame count.
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
				val = silenceValue
			}
			record = append(record, strconv.FormatFloat(val, 'f', 6, 64))
		}

		if err := w.Write(record); err != nil {
			return fmt.Errorf("failed to write record for frame %d: %w", frameIdx, err)
		}
	}
	return nil
}

func (lm *LoudnessMap) ToCSV() (string, error) {
	var buf strings.Builder
	if err := lm.WriteWideCSV(&buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// columnMapping describes one output column read from the CSV.
type columnMapping struct {
	streamIdx string
	metricKey string // e.g. "1.RMS_peak"
}

// parseCSVHeader validates the header row and extracts column mappings.
func parseCSVHeader(header []string) ([]columnMapping, error) {
	if len(header) < 2 || header[0] != "frame" || header[1] != "time" {
		return nil, fmt.Errorf("некорректный заголовок: ожидались 'frame', 'time', а получено %v", header)
	}

	cols := make([]columnMapping, 0, len(header)-2)
	for i := 2; i < len(header); i++ {
		colName := header[i]
		// expected format: "stream_<idx>_<metricKey>"
		parts := strings.SplitN(colName, "_", 3)
		if len(parts) != 3 || parts[0] != "stream" {
			return nil, fmt.Errorf("неверный формат колонки %q: ожидается 'stream_<idx>_<metricKey>'", colName)
		}
		cols = append(cols, columnMapping{
			streamIdx: parts[1],
			metricKey: parts[2],
		})
	}
	return cols, nil
}

// populateFromCSVRow reads one data record and appends its values to the map.
func (lm *LoudnessMap) populateFromCSVRow(record []string, cols []columnMapping) error {
	if len(record) < 2 {
		return fmt.Errorf("строка слишком короткая: %v", record)
	}

	for idx, col := range cols {
		var val float64 = silenceValue
		if idx+2 < len(record) {
			var err error
			val, err = parseValue(record[idx+2])
			if err != nil {
				return fmt.Errorf("ошибка парсинга значения в строке %v, колонка %q: %w", record, col.metricKey, err)
			}
		}
		lm.appendData(col.streamIdx, col.metricKey, val)
	}
	return nil
}

// LoudnessMapFromCSV reconstructs a LoudnessMap from a CSV file.
func LoudnessMapFromCSV(csvPath string) (*LoudnessMap, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл %s: %w", csvPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // allow variable number of fields

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения заголовка: %w", err)
	}

	columns, err := parseCSVHeader(header)
	if err != nil {
		return nil, err
	}

	lm := NewLoudnessMap()
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("ошибка чтения строки: %w", err)
		}

		if err := lm.populateFromCSVRow(record, columns); err != nil {
			return nil, err
		}
	}

	return lm, nil
}
