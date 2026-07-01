package mediagroup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
	"github.com/Galdoba/ffquery/pkg/ffprobe"
	"github.com/Galdoba/ffquery/pkg/media/streammap"
	"github.com/Galdoba/ffquery/pkg/progress"
)

const (
	minimalIntervalSamples        = 200
	defaultIntervalDurationFactor = 10

	progressFileKey = "progress"
	csvFileKey      = "csv"
	astatsCSVKey    = "astatsCSV"

	progressTickInterval = 500 * time.Millisecond
	progressPrefix       = "out_time_us="
	microsPerSecond      = 1_000_000
	percentMultiplier    = 100.0
)

var usePipes = runtime.GOOS != "windows"

type audioScanOutputs struct {
	Mode         string
	Paths        map[string]string
	PipeReaders  map[string]io.ReadCloser
	closeWriters func()
	CSVPath      string
}

type scanConfig struct {
	Streams         []AudioStreamInfo
	PerChannel      []filters.AstatMeasure
	Overall         []filters.AstatMeasure
	Dir             string
	Prefix          string
	CSVPath         string
	InputPath       string
	progressTracker *progress.Tracker
	cmd             *exec.Cmd
	output          *audioScanOutputs
}

type ScanOption func(*scanConfig) error

func WithPerChannelMeasures(measures ...filters.AstatMeasure) ScanOption {
	return func(scan *scanConfig) error {
		scan.PerChannel = measures
		return nil
	}
}

func WithOverallMeasures(measures ...filters.AstatMeasure) ScanOption {
	return func(scan *scanConfig) error {
		scan.Overall = measures
		return nil
	}
}

// WithProgressTracker sets an optional progress tracker. If opts are empty, no tracker is used.
func WithProgressTracker(opts ...progress.Option) ScanOption {
	return func(scan *scanConfig) error {
		if len(opts) > 0 {
			scan.progressTracker = progress.NewTracker(opts...)
		}
		return nil
	}
}

// ScanAstats runs an astats loudness scan on all audio streams of the media file.
func (m *Media) ScanAstats(ctx context.Context, opts ...ScanOption) error {
	scanCfg, err := m.generateLoudnessStatsScanCommand(opts...)
	if err != nil {
		return fmt.Errorf("failed to generate ffmpeg command: %w", err)
	}
	return m.executeAudioScanCommand(ctx, scanCfg)
}

func (m *Media) generateLoudnessStatsScanCommand(options ...ScanOption) (*scanConfig, error) {
	streams, err := m.collectAudioStreamInfo()
	if err != nil {
		return nil, fmt.Errorf("collecting audio streams: %w", err)
	}
	if len(streams) == 0 {
		return nil, errors.New("at least one audio stream is required")
	}

	dir := filepath.Dir(m.Path)
	prefix := strings.TrimSuffix(filepath.Base(m.Path), filepath.Ext(m.Path))
	csvPath := filepath.Join(dir, prefix+".AstatsScan.csv")

	cfg := &scanConfig{
		Streams:   streams,
		Dir:       dir,
		Prefix:    prefix,
		CSVPath:   csvPath,
		InputPath: m.Path,
	}

	for _, set := range options {
		if err := set(cfg); err != nil {
			return nil, err
		}
	}

	if usePipes {
		if err := cfg.generatePipeCommand(); err != nil {
			return nil, fmt.Errorf("failed to generate cmd: %w", err)
		}
	} else {
		if err := cfg.generateFileCommand(); err != nil {
			return nil, fmt.Errorf("failed to generate cmd: %w", err)
		}
	}

	return cfg, nil
}

// collectAudioStreamInfo builds a slice of AudioStreamInfo for the media's audio streams.
func (m *Media) collectAudioStreamInfo() ([]AudioStreamInfo, error) {
	var asi []AudioStreamInfo
	for i, a := range m.Audio {
		as := AudioStreamInfo{
			Index:         i,
			ChannelLayout: a.raw.ChannelLayout,
			Channels:      setChannelTags(a.raw),
		}
		if len(as.Channels) == 0 {
			return nil, fmt.Errorf("unknown channel layout for audio %d of %s: %q", i, m.Path, as.ChannelLayout)
		}
		intervalSamples := a.raw.SampleRateHz() / defaultIntervalDurationFactor
		if intervalSamples < minimalIntervalSamples {
			return nil, fmt.Errorf("sample rate for audio %d of %s is too low: %d Hz", i, m.Path, a.raw.SampleRateHz())
		}
		as.IntervalSamples = intervalSamples
		asi = append(asi, as)
	}
	return asi, nil
}

// AudioStreamInfo holds metadata needed for astats scanning of one audio stream.
type AudioStreamInfo struct {
	Index           int
	ChannelLayout   string
	Channels        []string
	IntervalSamples int
}

// ChannelNames maps known ffmpeg channel layouts to individual channel labels.
var ChannelNames = map[string][]string{
	"mono":      {"m"},
	"stereo":    {"L", "R"},
	"5.0":       {"L", "R", "C", "LB", "RB"},
	"5.1":       {"L", "R", "C", "LFE", "LB", "RB"},
	"5.1(side)": {"L", "R", "C", "LFE", "LS", "RS"},
	"6.1":       {"L", "R", "C", "LFE", "LB", "RB", "BC"},
	"7.1":       {"L", "R", "C", "LFE", "LB", "RB", "LS", "RS"},
}

// setChannelTags returns channel labels for the given stream, falling back to numbered labels.
func setChannelTags(r ffprobe.Stream) []string {
	chans := ChannelNames[r.ChannelLayout]
	if len(chans) > 0 {
		return chans
	}
	for i := 1; i <= r.Channels; i++ {
		chans = append(chans, fmt.Sprintf("%dch", i))
	}
	return chans
}

// executeAudioScanCommand dispatches to the correct execution mode.
func (m *Media) executeAudioScanCommand(ctx context.Context, scan *scanConfig) error {
	if scan.output.Mode == "pipes" {
		return m.executePipeScan(ctx, scan)
	}
	return m.executeFileScan(ctx, scan)
}

func parseOutTimeUs(line string) (int64, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, progressPrefix) {
		return -1, fmt.Errorf("not a progress time line")
	}
	return strconv.ParseInt(line[len(progressPrefix):], 10, 64)
}

func usToPercent(us int64, totalSeconds float64) float64 {
	if totalSeconds <= 0 {
		return 0
	}
	return (float64(us) / microsPerSecond / totalSeconds) * percentMultiplier
}

func (m *Media) writeWideCSVFromLoudnessMap(lm *streammap.LoudnessMap, csvPath string) error {
	f, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("creating CSV: %w", err)
	}
	defer f.Close()

	if err := lm.WriteWideCSV(f); err != nil {
		return fmt.Errorf("writing CSV: %w", err)
	}

	if m.Meta == nil {
		m.Meta = make(map[string]string)
	}
	m.Meta[astatsCSVKey] = csvPath
	return nil
}
