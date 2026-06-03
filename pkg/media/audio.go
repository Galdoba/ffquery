package mediagroup

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
)

const (
	minimalIntervalSamples        = 200
	defaultIntervalDurationFactor = 10
)

func (m *Media) ScanRmsLevels(audio ...int) error {
	// строим команду
	asi, err := m.collectAudioStreamInfo(audio...)
	if err != nil {
		return fmt.Errorf("failed to collect audio stream info: %w", err)
	}
	cmd, paths, err := generateLoudnessStatsScanCommand(m.Path, asi, filepath.Dir(m.Path))
	fmt.Println(cmd)
	for k, v := range paths {
		fmt.Println("===")
		fmt.Println(k, ":", v)
	}
	// выполняем команду
	// парсим файлы
	// записываем данные в аудио потоки
	err = m.ExecScanRMS(asCommand(cmd))
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
	"5.0":       {"L", "R", "C", "Lb", "Rb"},
	"5.1":       {"L", "R", "C", "Lfe", "Lb", "Rb"},
	"5.1(side)": {"L", "R", "C", "Lfe", "Ls", "Rs"},
	"6.1":       {"L", "R", "C", "Lfe", "Lb", "Rb", "BC"},
	"7.1":       {"L", "R", "C", "Lfe", "Lb", "Rb", "Ls", "Rs"},
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

func asCommand(cm string) *exec.Cmd {
	cm = strings.ReplaceAll(cm, "ffmpeg ", "ffmpeg -nostats -v error ")
	cm = strings.ReplaceAll(cm, `"`, "")
	re := regexp.MustCompile(`file='.*?'\[`)
	cm = re.ReplaceAllString(cm, "file=-[")
	args := strings.Split(cm, " ")
	return exec.Command(args[0], args[1:]...)
}

func (m *Media) ExecScanRMS(cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	// Захват stderr (чтобы не потерять ошибки ffmpeg)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// Чтение stdout и stderr параллельно
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "frame") || strings.Contains(line, "RMS_level") || strings.Contains(line, "Peak_level") {
				fmt.Println(line)
			}
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintln(os.Stderr, "FFmpeg:", scanner.Text())
		}
	}()

	wg.Wait()
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}

func generateLoudnessStatsScanCommand(inputFile string, streams []AudioStreamInfo, outputDir string) (string, map[string]string, error) {
	if len(streams) == 0 {
		return "", nil, errors.New("at least one audio stream must be provided")
	}

	var filterParts []string
	var mapParts []string
	outputFiles := make(map[string]string)
	outNamePrefix := filepath.Base(inputFile)
	outNamePrefix = strings.TrimSuffix(outNamePrefix, filepath.Ext(outNamePrefix))

	for _, s := range streams {
		streamTag := fmt.Sprintf("stream_%d", s.Index)
		fileName := fmt.Sprintf("%s_stream_%d.txt", outNamePrefix, s.Index)
		filePath := filepath.Join(outputDir, fileName)
		outputFiles[fmt.Sprintf("%d", s.Index)] = filePath
		filterParts = append(filterParts,
			fmt.Sprintf("[0:a:%d]asetnsamples=%d,astats=metadata=1:reset=1,ametadata=mode=add:key=frame_end:value=%s,ametadata=mode=print:file='%s'[%s]",
				s.Index, s.IntervalSamples, streamTag, filePath, streamTag))
		mapParts = append(mapParts, fmt.Sprintf("-map [%s]", streamTag))
	}

	filterComplex := strings.Join(filterParts, ";")
	mapArgs := strings.Join(mapParts, " ")

	cmd := fmt.Sprintf("ffmpeg -i %s -filter_complex \"%s\" %s -f null -",
		inputFile, filterComplex, mapArgs)
	cmd = strings.ReplaceAll(cmd, "\\", "/")
	for k := range outputFiles {
		outputFiles[k] = filepath.ToSlash(outputFiles[k])
	}

	return cmd, outputFiles, nil
}

func testCMD() {
	cmd := exec.Command("ffmpeg", "-i", "input")
	cmd.Args = append(cmd.Args, "out")
	fmt.Println(cmd.Args)
}
