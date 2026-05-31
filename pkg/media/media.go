package mediagroup

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
)

type MediaGroup struct {
	Tags       map[string]string
	MediaFiles []*Media
}

type Media struct {
	Path string
	Raw  ffprobe.RawData
}

type VStream struct {
	FileIndex         int
	StreamIndex       int
	raw               ffprobe.Stream
	InterlaceDetected bool
}
type AStream struct {
	FileIndex   int
	StreamIndex int
	raw         ffprobe.Stream
	rma         [][]float64 //csv data for RMA
	lufs        [][]float64 //csv data for LUFS
}
type DStream struct{}
type SStream struct{}

func newMedia(path string) (*Media, error) {
	m := Media{}
	r, err := ffprobe.NewRawData(path)
	if err != nil {
		return nil, fmt.Errorf("failed to colect raw data: %w", err)
	}
	m.Raw = r
	m.Path = path
	return &m, nil
}

// New wil create new mediagroup from files.
//
// At least one file must be provided. All files must be located in the same directory.
func New(paths ...string) (*MediaGroup, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("mediagroup must consist of at least 1 file")
	}
	dir := filepath.Dir(paths[0])
	files := make(map[string]int, len(paths))
	for _, p := range paths {
		if filepath.Dir(p) != dir {
			return nil, fmt.Errorf("files must be located in the same directory: file %q is not in %q", p, dir)
		}
		files[p]++
		if files[p] != 1 {
			return nil, fmt.Errorf("duplicated arguments provided: %q", p)
		}
	}
	mg := MediaGroup{}
	slices.Sort(paths)
	for _, p := range paths {
		m, err := newMedia(p)
		if err != nil {
			return nil, err
		}
		mg.MediaFiles = append(mg.MediaFiles, m)
	}

	return &mg, nil
}

func (m *Media) ScanAudioCommand(indexes ...int) (string, error) {
	if m.Raw.Format == nil || m.Raw.Format.Filename == "" {
		return "", fmt.Errorf("no input file information")
	}

	// Collect all audio streams with their global and audio index
	type audioInfo struct {
		globalIndex int
		audioIndex  int // 0-based index among audio streams
		stream      ffprobe.Stream
	}
	var audioStreams []audioInfo
	for i, s := range m.Raw.Streams {
		if s.CodecType == ffprobe.StreamTypeAudio {
			audioStreams = append(audioStreams, audioInfo{
				globalIndex: s.Index,
				audioIndex:  len(audioStreams),
				stream:      m.Raw.Streams[i],
			})
		}
	}
	if len(audioStreams) == 0 {
		return "", fmt.Errorf("no audio streams found in file")
	}

	// If no indexes specified, scan all audio streams
	if len(indexes) == 0 {
		indexes = make([]int, len(audioStreams))
		for i, a := range audioStreams {
			indexes[i] = a.globalIndex
		}
	}

	// Build list of streams to process
	var toScan []audioInfo
	for _, idx := range indexes {
		found := false
		for _, a := range audioStreams {
			if a.globalIndex == idx {
				toScan = append(toScan, a)
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("audio stream with global index %d not found or not audio", idx)
		}
	}

	// Determine video FPS
	videoFPS := 25.0
	for _, s := range m.Raw.Streams {
		if s.CodecType == ffprobe.StreamTypeVideo {
			if fps := s.FPS(); fps > 0 {
				videoFPS = fps
				break
			}
		}
	}

	basename := filepath.Base(m.Raw.Format.Filename)
	if ext := filepath.Ext(basename); ext != "" {
		basename = basename[:len(basename)-len(ext)]
	}

	var filterParts, mapParts []string

	for _, aud := range toScan {
		s := aud.stream
		sampleRate := s.SampleRateHz()
		if sampleRate <= 0 {
			return "", fmt.Errorf("stream %d: invalid sample rate", aud.globalIndex)
		}
		channels := s.Channels
		if channels <= 0 {
			return "", fmt.Errorf("stream %d: invalid channel count", aud.globalIndex)
		}
		layout := s.ChannelLayout
		if layout == "" {
			switch channels {
			case 1:
				layout = "mono"
			case 2:
				layout = "stereo"
			case 6:
				layout = "5.1"
			case 8:
				layout = "7.1"
			default:
				return "", fmt.Errorf("stream %d: unsupported channel layout (%d ch)", aud.globalIndex, channels)
			}
		}

		samplesPerFrame := int(math.Round(float64(sampleRate) / videoFPS))
		if samplesPerFrame < 1 {
			samplesPerFrame = 1
		}

		globalIdx := aud.globalIndex
		audioIdx := aud.audioIndex
		prefix := basename + "_stream" + strconv.Itoa(globalIdx)

		lblMix := fmt.Sprintf("a_lufs_mix_%d", globalIdx)
		lblRmsInt := fmt.Sprintf("a_rms_int_%d", globalIdx)
		lblRmsOvr := fmt.Sprintf("a_rms_ovr_%d", globalIdx)
		lblChSplit := fmt.Sprintf("a_lufs_ch_%d", globalIdx)

		nullMix := fmt.Sprintf("null_l_%d", globalIdx)
		nullRmsInt := fmt.Sprintf("null_ri_%d", globalIdx)
		nullRmsOvr := fmt.Sprintf("null_ro_%d", globalIdx)

		// asplit – используем порядковый номер аудиопотока
		splitLine := fmt.Sprintf("[0:a:%d]asplit=4 [%s] [%s] [%s] [%s]",
			audioIdx, lblMix, lblRmsInt, lblRmsOvr, lblChSplit)
		filterParts = append(filterParts, splitLine)

		// LUFS mix
		mixLine := fmt.Sprintf("[%s] ebur128=video=0:meter=18:metadata=1, ametadata=mode=print:file=%s_ebur128_mix.txt [%s]",
			lblMix, prefix, nullMix)
		filterParts = append(filterParts, mixLine)

		// RMS intervals
		rmsIntLine := fmt.Sprintf("[%s] asetnsamples=%d, astats=metadata=1:reset=1, ametadata=mode=print:file=%s_rms_intervals.txt [%s]",
			lblRmsInt, samplesPerFrame, prefix, nullRmsInt)
		filterParts = append(filterParts, rmsIntLine)

		// RMS overall
		rmsOvrLine := fmt.Sprintf("[%s] astats=metadata=1:reset=0, ametadata=mode=print:file=%s_rms_overall.txt [%s]",
			lblRmsOvr, prefix, nullRmsOvr)
		filterParts = append(filterParts, rmsOvrLine)

		// Channels split
		chLabels := make([]string, channels)
		nullChLabels := make([]string, channels)
		for ch := 0; ch < channels; ch++ {
			chLabels[ch] = fmt.Sprintf("ch%d_%d", ch+1, globalIdx)
			nullChLabels[ch] = fmt.Sprintf("null_ch%d_%d", ch+1, globalIdx)
		}
		chLabelsBracketed := make([]string, channels)
		for i, label := range chLabels {
			chLabelsBracketed[i] = "[" + label + "]"
		}
		chSplitLine := fmt.Sprintf("[%s] channelsplit=channel_layout=%s %s",
			lblChSplit, layout, strings.Join(chLabelsBracketed, " "))
		filterParts = append(filterParts, chSplitLine)

		for ch := 0; ch < channels; ch++ {
			chLine := fmt.Sprintf("[%s] ebur128=video=0:meter=18:metadata=1, ametadata=mode=print:file=%s_ebur128_ch%d.txt [%s]",
				chLabels[ch], prefix, ch+1, nullChLabels[ch])
			filterParts = append(filterParts, chLine)
		}

		mapParts = append(mapParts, fmt.Sprintf("[%s]", nullMix))
		mapParts = append(mapParts, fmt.Sprintf("[%s]", nullRmsInt))
		mapParts = append(mapParts, fmt.Sprintf("[%s]", nullRmsOvr))
		for _, nl := range nullChLabels {
			mapParts = append(mapParts, fmt.Sprintf("[%s]", nl))
		}
	}

	filterGraph := strings.Join(filterParts, "; ")
	mapList := strings.Join(mapParts, " -map ")
	cmd := fmt.Sprintf(`ffmpeg -i "%s" -filter_complex "%s" -map %s -f null NUL`,
		m.Raw.Format.Filename, filterGraph, mapList)
	return cmd, nil
}
