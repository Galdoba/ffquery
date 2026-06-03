package mediagroup

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
)

type MediaGroup struct {
	Tags       map[string]string
	MediaFiles []*Media
}

type Media struct {
	Path      string
	Raw       ffprobe.RawData
	Video     []VStream
	Audio     []AStream
	Data      []DStream
	Subtitles []SStream
}

type VStream struct {
	FileIndex         int
	StreamIndex       int
	Position          int
	raw               ffprobe.Stream
	InterlaceDetected bool
}
type AStream struct {
	FileIndex   int
	StreamIndex int
	Position    int
	raw         ffprobe.Stream
	rmaLevels   [][]float64 //csv data for RMA
}
type DStream struct {
	FileIndex   int
	StreamIndex int
	Position    int
	raw         ffprobe.Stream
}
type SStream struct {
	FileIndex   int
	StreamIndex int
	Position    int
	raw         ffprobe.Stream
}

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
		if strings.Contains(p, " ") {
			return nil, fmt.Errorf("filepath must not contain spaces: %q", p)
		}
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

	//ingest files
	for fileIndex, p := range paths {
		m, err := processFile(fileIndex, p)
		if err != nil {
			return nil, err
		}
		mg.MediaFiles = append(mg.MediaFiles, m)
	}

	return &mg, nil
}

// processFile creates a MediaFile from a given path and file index.
// It parses the file using ffprobe and classifies its streams into video, audio, data, and subtitles.
func processFile(fileIndex int, path string) (*Media, error) {
	m, err := newMedia(path)
	if err != nil {
		return nil, err
	}
	for streamIndex, s := range m.Raw.Streams {
		fmt.Println("add", s.CodecType)
		switch s.CodecType {
		case ffprobe.StreamTypeVideo:
			m.Video = append(m.Video, VStream{
				FileIndex:   fileIndex,
				StreamIndex: len(m.Video),
				Position:    streamIndex,
				raw:         s,
			})
		case ffprobe.StreamTypeAudio:
			m.Audio = append(m.Audio, AStream{
				FileIndex:   fileIndex,
				StreamIndex: len(m.Audio),
				Position:    streamIndex,
				raw:         s,
			})
		case ffprobe.StreamTypeData:
			m.Data = append(m.Data, DStream{
				FileIndex:   fileIndex,
				StreamIndex: len(m.Data),
				Position:    streamIndex,
				raw:         s,
			})
		case ffprobe.StreamTypeSubtitle:
			m.Subtitles = append(m.Subtitles, SStream{
				FileIndex:   fileIndex,
				StreamIndex: len(m.Subtitles),
				Position:    streamIndex,
				raw:         s,
			})
		default:
			return nil, fmt.Errorf("unknown codec type for stream %q: %q",
				fmt.Sprintf("[%d:%d]", fileIndex, streamIndex), s.CodecType)
		}
	}
	return m, nil
}
