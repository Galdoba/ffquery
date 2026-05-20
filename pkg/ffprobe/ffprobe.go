package ffprobe

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Type definitions
// ---------------------------------------------------------------------------

// RawData represents the root object of the full ffprobe output in JSON format.
// Depending on the options used (e.g., -show_streams, -show_format),
// some fields may be missing. It is recommended to use pointers (`*`) or
// check for the presence of a field before accessing it.
type RawData struct {
	// source is the path to the file from which the data was extracted.
	source string `json:"-"`
	// ProgramVersion contains information about the ffprobe version.
	ProgramVersion *ProgramVersion `json:"program_version,omitempty"`
	// LibraryVersions contains a list of library versions used by ffprobe.
	LibraryVersions []LibraryVersion `json:"library_versions,omitempty"`
	// PixelFormats contains a list of supported pixel formats.
	PixelFormats []PixelFormat `json:"pixel_formats,omitempty"`
	// Packets contains information about data packets of the media file.
	Packets []Packet `json:"packets,omitempty"`
	// Frames contains information about frames (video/audio/subtitles).
	Frames []Frame `json:"frames,omitempty"`
	// PacketsAndFrames contains combined information about packets and frames.
	PacketsAndFrames []PacketOrFrame `json:"packets_and_frames,omitempty"`
	// Programs contains information about programs (usually in MPEG-TS streams).
	Programs []Program `json:"programs,omitempty"`
	// Streams contains information about all streams (video, audio, subtitles, data).
	Streams []Stream `json:"streams,omitempty"`
	// Format contains information about the format (container) of the media file.
	Format *Format `json:"format,omitempty"`
	// Chapters contains information about chapters in the media file.
	Chapters []Chapter `json:"chapters,omitempty"`
	// Error contains information about an error that occurred during analysis.
	Error *ProbeError `json:"error,omitempty"`
}

// ProgramVersion describes the version of ffprobe.
type ProgramVersion struct {
	Version       string `json:"version"`
	Copyright     string `json:"copyright"`
	CompilerIdent string `json:"compiler_ident"`
	Configuration string `json:"configuration"`
}

// LibraryVersion describes the version of a single library.
type LibraryVersion struct {
	Name    string `json:"name"`
	Major   int    `json:"major"`
	Minor   int    `json:"minor"`
	Micro   int    `json:"micro"`
	Version int    `json:"version"`
	Ident   string `json:"ident"`
}

// PixelFormat describes a pixel format.
type PixelFormat struct {
	Name         string `json:"name"`
	NbComponents int    `json:"nb_components"`
	Log2ChromaW  int    `json:"log2_chroma_w"`
	Log2ChromaH  int    `json:"log2_chroma_h"`
	Bpp          int    `json:"bpp"`
	Flags        string `json:"flags"`
}

// Packet describes a single data packet.
type Packet struct {
	CodecType    string     `json:"codec_type"`
	StreamIndex  int        `json:"stream_index"`
	Pts          int64      `json:"pts"`
	PtsTime      string     `json:"pts_time"`
	Dts          int64      `json:"dts"`
	DtsTime      string     `json:"dts_time"`
	Duration     int64      `json:"duration"`
	DurationTime string     `json:"duration_time"`
	Size         string     `json:"size"`
	Pos          string     `json:"pos"`
	Flags        string     `json:"flags"`
	SideDataList []SideData `json:"side_data_list,omitempty"`
}

// SideData describes side data of a packet.
type SideData struct {
	SideDataType string `json:"side_data_type"`
	Size         int    `json:"size"`
}

// Frame describes a single frame (video, audio, or subtitle).
type Frame struct {
	MediaType               string `json:"media_type"`
	StreamIndex             int    `json:"stream_index"`
	KeyFrame                int    `json:"key_frame"`
	Pts                     int64  `json:"pts"`
	PtsTime                 string `json:"pts_time"`
	PktDts                  int64  `json:"pkt_dts"`
	PktDtsTime              string `json:"pkt_dts_time"`
	BestEffortTimestamp     int64  `json:"best_effort_timestamp"`
	BestEffortTimestampTime string `json:"best_effort_timestamp_time"`
	PktPos                  string `json:"pkt_pos"`
	PktDuration             int64  `json:"pkt_duration"`
	PktDurationTime         string `json:"pkt_duration_time"`
	PktSize                 string `json:"pkt_size"`

	// Video-specific
	Width             int    `json:"width,omitempty"`
	Height            int    `json:"height,omitempty"`
	PixFmt            string `json:"pix_fmt,omitempty"`
	SampleAspectRatio string `json:"sample_aspect_ratio,omitempty"`
	PictType          string `json:"pict_type,omitempty"`
	CodedPictNumber   int    `json:"coded_pict_number,omitempty"`
	DisplayPictNumber int    `json:"display_pict_number,omitempty"`
	InterlacedFrame   int    `json:"interlaced_frame,omitempty"`
	TopFieldFirst     int    `json:"top_field_first,omitempty"`
	RepeatPict        int    `json:"repeat_pict,omitempty"`

	// Audio-specific
	SampleFmt     string `json:"sample_fmt,omitempty"`
	SampleRate    int    `json:"sample_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
	NbSamples     int    `json:"nb_samples,omitempty"`
}

// PacketOrFrame can be either a packet or a frame. Used in packs_and_frames.
type PacketOrFrame struct {
	Type string `json:"type"` // "packet" or "frame"
}

// Program describes a program (usually in MPEG-TS).
type Program struct {
	ProgramID  int               `json:"program_id"`
	ProgramNum int               `json:"program_num"`
	NbStreams  int               `json:"nb_streams"`
	PmtPID     int               `json:"pmt_pid"`
	PcrPID     int               `json:"pcr_pid"`
	StartPts   int64             `json:"start_pts"`
	StartTime  string            `json:"start_time"`
	EndPts     int64             `json:"end_pts"`
	EndTime    string            `json:"end_time"`
	Streams    []Stream          `json:"streams,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

// Format describes the format (container) of a media file.
type Format struct {
	Filename       string            `json:"filename"`
	NbStreams      int               `json:"nb_streams"`
	NbPrograms     int               `json:"nb_programs"`
	FormatName     string            `json:"format_name"`
	FormatLongName string            `json:"format_long_name"`
	StartTime      string            `json:"start_time"`
	Duration       string            `json:"duration"`
	Size           string            `json:"size"`
	BitRate        string            `json:"bit_rate"`
	ProbeScore     int               `json:"probe_score"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// Chapter describes a chapter in a media file.
type Chapter struct {
	ID        int               `json:"id"`
	TimeBase  string            `json:"time_base"`
	Start     int64             `json:"start"`
	StartTime string            `json:"start_time"`
	End       int64             `json:"end"`
	EndTime   string            `json:"end_time"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// ProbeError contains information about an error that occurred while analyzing a file.
type ProbeError struct {
	Code   int    `json:"code"`
	String string `json:"string"`
}

// Stream describes a single media stream (video, audio, subtitles, data).
type Stream struct {
	Index          int    `json:"index"`
	CodecName      string `json:"codec_name,omitempty"`
	CodecLongName  string `json:"codec_long_name,omitempty"`
	Profile        string `json:"profile,omitempty"`
	CodecType      string `json:"codec_type,omitempty"`
	CodecTimeBase  string `json:"codec_time_base,omitempty"`
	CodecTagString string `json:"codec_tag_string"`
	CodecTag       string `json:"codec_tag"`

	// Video-specific
	Width              int    `json:"width,omitempty"`
	Height             int    `json:"height,omitempty"`
	CodedWidth         int    `json:"coded_width,omitempty"`
	CodedHeight        int    `json:"coded_height,omitempty"`
	ClosedCaptions     int    `json:"closed_captions,omitempty"`
	HasBFrames         int    `json:"has_b_frames,omitempty"`
	SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
	PixFmt             string `json:"pix_fmt,omitempty"`
	Level              int    `json:"level,omitempty"`
	ColorRange         string `json:"color_range,omitempty"`
	ColorSpace         string `json:"color_space,omitempty"`
	ColorTransfer      string `json:"color_transfer,omitempty"`
	ColorPrimaries     string `json:"color_primaries,omitempty"`
	ChromaLocation     string `json:"chroma_location,omitempty"`
	FieldOrder         string `json:"field_order,omitempty"`
	Refs               int    `json:"refs,omitempty"`
	IsAvc              string `json:"is_avc,omitempty"`
	NalLengthSize      string `json:"nal_length_size,omitempty"`
	BitsPerRawSample   string `json:"bits_per_raw_sample,omitempty"`

	// Audio-specific
	SampleFmt     string `json:"sample_fmt,omitempty"`
	SampleRate    string `json:"sample_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
	BitsPerSample int    `json:"bits_per_sample,omitempty"`

	// Common
	ID            string `json:"id,omitempty"`
	RFrameRate    string `json:"r_frame_rate,omitempty"`
	AvgFrameRate  string `json:"avg_frame_rate,omitempty"`
	TimeBase      string `json:"time_base,omitempty"`
	StartPts      int64  `json:"start_pts,omitempty"`
	StartTime     string `json:"start_time,omitempty"`
	DurationTS    int64  `json:"duration_ts,omitempty"`
	Duration      string `json:"duration,omitempty"`
	BitRate       string `json:"bit_rate,omitempty"`
	MaxBitRate    string `json:"max_bit_rate,omitempty"`
	NbFrames      string `json:"nb_frames,omitempty"`
	NbReadFrames  string `json:"nb_read_frames,omitempty"`
	NbReadPackets string `json:"nb_read_packets,omitempty"`

	Disposition map[string]int    `json:"disposition,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// ffprobe command and default scan arguments.
const (
	ffprobeCmd = "ffprobe"
)

var ffprobeScanArgs = []string{
	"-v", "quiet",
	"-print_format", "json",
	"-show_streams",
	"-show_format",
	"-show_chapters",
	"-show_error",
	"-show_data",
}

// Commonly used tag keys and stream type identifiers.
const (
	// TagLanguage    = "language"
	// TagTitle       = "title"
	// TagHandlerName = "handler_name"

	StreamTypeVideo    = "video"
	StreamTypeAudio    = "audio"
	StreamTypeSubtitle = "subtitle"
	StreamTypeData     = "data"
)

// Render field names (used for filtering output).
const (
	// Top-level sections
	FieldName      = "Name"
	FieldVideo     = "Video"
	FieldAudio     = "Audio"
	FieldSubtitles = "Subtitles"
	FieldData      = "Data"

	// Common stream fields
	FieldIndex         = "Index"
	FieldTypeIndex     = "TypeIndex"
	FieldCodec         = "Codec"
	FieldCodecLongName = "CodecLongName"
	FieldProfile       = "Profile"
	FieldBitRate       = "BitRate"
	FieldDuration      = "Duration"

	// Video-specific
	FieldWidth        = "Width"
	FieldHeight       = "Height"
	FieldPixFmt       = "PixFmt"
	FieldFrameRate    = "FrameRate"
	FieldAvgFrameRate = "AvgFrameRate"
	FieldNbFrames     = "NbFrames"

	// Audio-specific
	FieldChannelLayout = "ChannelLayout"
	FieldSampleRate    = "SampleRate"
	FieldChannels      = "Channels"

	// Data-specific
	FieldCodecTag = "CodecTag"

	// Additional fields (shown when no filter is active)
	FieldCodecType          = "CodecType"
	FieldCodecTimeBase      = "CodecTimeBase"
	FieldCodecTagHex        = "CodecTagHex"
	FieldCodedWidth         = "CodedWidth"
	FieldCodedHeight        = "CodedHeight"
	FieldClosedCaptions     = "ClosedCaptions"
	FieldHasBFrames         = "HasBFrames"
	FieldSampleAspectRatio  = "SampleAspectRatio"
	FieldDisplayAspectRatio = "DisplayAspectRatio"
	FieldLevel              = "Level"
	FieldColorRange         = "ColorRange"
	FieldColorSpace         = "ColorSpace"
	FieldColorTransfer      = "ColorTransfer"
	FieldColorPrimaries     = "ColorPrimaries"
	FieldChromaLocation     = "ChromaLocation"
	FieldFieldOrder         = "FieldOrder"
	FieldRefs               = "Refs"
	FieldIsAvc              = "IsAvc"
	FieldNalLengthSize      = "NalLengthSize"
	FieldBitsPerRawSample   = "BitsPerRawSample"
	FieldSampleFmt          = "SampleFmt"
	FieldBitsPerSample      = "BitsPerSample"
	FieldID                 = "ID"
	FieldTimeBase           = "TimeBase"
	FieldStartPts           = "StartPts"
	FieldStartTime          = "StartTime"
	FieldDurationTS         = "DurationTS"
	FieldMaxBitRate         = "MaxBitRate"
	FieldNbReadFrames       = "NbReadFrames"
	FieldNbReadPackets      = "NbReadPackets"

	// Format fields
	FieldFormatName     = "FormatName"
	FieldFormatLongName = "FormatLongName"
	FieldSize           = "Size"
	FieldProbeScore     = "ProbeScore"
	FieldNbStreams      = "NbStreams"
	FieldNbPrograms     = "NbPrograms"
	FieldMapIndex       = "MapIndex"

	// Meta fields
	FieldTags        = "Tags"
	FieldDisposition = "Disposition"

	// Formatting
	indent     = "  "
	separation = ": "
)

// ---------------------------------------------------------------------------
// Predefined field slices (currently unused but kept for possible external
// reference or future use).
// ---------------------------------------------------------------------------

var (
	videoFieldNames    = []string{FieldIndex, FieldTypeIndex, FieldCodec, FieldCodecLongName, FieldProfile, FieldWidth, FieldHeight, FieldPixFmt, FieldFrameRate, FieldAvgFrameRate, FieldBitRate, FieldDuration, FieldNbFrames}
	audioFieldNames    = []string{FieldIndex, FieldTypeIndex, FieldCodec, FieldCodecLongName, FieldProfile, FieldChannelLayout, FieldSampleRate, FieldChannels, FieldBitRate, FieldDuration}
	subtitleFieldNames = []string{FieldIndex, FieldTypeIndex, FieldCodec}
	dataFieldNames     = []string{FieldIndex, FieldTypeIndex, FieldCodecTag}
)

// ---------------------------------------------------------------------------
// Internal helper type for Render
// ---------------------------------------------------------------------------

// streamWithTypeIndex pairs a Stream with its index within its type group.
type streamWithTypeIndex struct {
	Stream
	TypeIndex int
	MapIndex  string
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewRawData creates a new RawData by running ffprobe on the given file path.
func NewRawData(path string) (*RawData, error) {
	ffprobePath, err := exec.LookPath(ffprobeCmd)
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found: %w", err)
	}
	if _, err := os.OpenFile(path, os.O_RDONLY, 0666); err != nil {
		return nil, fmt.Errorf("cannot read file %q: %w", path, err)
	}

	cmd := exec.Command(ffprobePath, append(ffprobeScanArgs, path)...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe %w\n%s", err, stdout.String())
	}

	var result RawData
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse RawData: %w\nraw output: %s", err, stdout.String())
	}

	result.source = path

	return &result, nil
}

// ---------------------------------------------------------------------------
// Render method and helpers
// ---------------------------------------------------------------------------

// Render returns a human-readable, indented representation of the media file
// described by the RawData. The optional keys argument specifies which fields
// should be included. If no keys are provided, all non-zero fields are rendered.
// Top-level categories (Video, Audio, etc.) are only shown if they are
// explicitly requested or if at least one of their fields is requested.
func (rd *RawData) Render(keys ...string) string {
	var b strings.Builder
	filter := make(map[string]bool, len(keys))
	for _, k := range keys {
		filter[k] = true
	}
	showAll := len(filter) == 0

	// File name
	if showAll || filter[FieldName] {
		name := ""
		if rd.Format != nil && rd.Format.Filename != "" {
			name = filepath.Base(rd.Format.Filename)
		} else {
			name = filepath.Base(rd.source)
		}
		if name != "" {
			b.WriteString(FieldName + separation + name + "\n")
		}
	}

	// Container (Format) fields
	if rd.Format != nil {
		b.WriteString("Format" + "\n")
		if (showAll || filter[FieldFormatName]) && rd.Format.FormatName != "" {
			b.WriteString(indent + FieldFormatName + separation + rd.Format.FormatName + "\n")
		}
		if (showAll || filter[FieldFormatLongName]) && rd.Format.FormatLongName != "" {
			b.WriteString(indent + FieldFormatLongName + separation + rd.Format.FormatLongName + "\n")
		}
		if (showAll || filter[FieldDuration]) && rd.Format.Duration != "" {
			b.WriteString(indent + FieldDuration + separation + rd.Format.Duration + "\n")
		}
		if (showAll || filter[FieldStartTime]) && rd.Format.StartTime != "" {
			b.WriteString(indent + FieldStartTime + separation + rd.Format.StartTime + "\n")
		}
		if (showAll || filter[FieldBitRate]) && rd.Format.BitRate != "" {
			b.WriteString(indent + FieldBitRate + separation + rd.Format.BitRate + "\n")
		}
		if (showAll || filter[FieldSize]) && rd.Format.Size != "" {
			b.WriteString(indent + FieldSize + separation + rd.Format.Size + "\n")
		}
		if showAll || filter[FieldProbeScore] {
			b.WriteString(indent + FieldProbeScore + separation + strconv.Itoa(rd.Format.ProbeScore) + "\n")
		}
		if showAll || filter[FieldNbStreams] {
			b.WriteString(indent + FieldNbStreams + separation + strconv.Itoa(rd.Format.NbStreams) + "\n")
		}
		if showAll || filter[FieldNbPrograms] {
			b.WriteString(indent + FieldNbPrograms + separation + strconv.Itoa(rd.Format.NbPrograms) + "\n")
		}
		if showAll && rd.Format.Tags != nil {
			b.WriteString(tagsToString(rd.Format.Tags))
		}
	}

	// Group streams by type
	var videos, audios, subtitles, datas []streamWithTypeIndex
	for _, s := range rd.Streams {
		switch s.CodecType {
		case StreamTypeVideo:
			i := len(videos)
			videos = append(videos, streamWithTypeIndex{s, i, fmt.Sprintf(":v:%d", i)})
		case StreamTypeAudio:
			i := len(audios)
			audios = append(audios, streamWithTypeIndex{s, i, fmt.Sprintf(":a:%d", i)})
		case StreamTypeSubtitle:
			i := len(subtitles)
			subtitles = append(subtitles, streamWithTypeIndex{s, i, fmt.Sprintf(":s:%d", i)})
		case StreamTypeData:
			i := len(datas)
			datas = append(datas, streamWithTypeIndex{s, i, fmt.Sprintf(":d:%d", i)})
		}
	}

	// Map of stream type index to printable name
	streamTypeMap := []string{"Video", "Audio", "Subtitle", "Data"}

	for i, typedStreams := range [][]streamWithTypeIndex{videos, audios, subtitles, datas} {
		if len(typedStreams) > 0 {
			b.WriteString(streamTypeMap[i] + "\n")
		}
		for _, stream := range typedStreams {
			if showAll || filter[FieldMapIndex] {
				b.WriteString("#Stream" + separation + fmt.Sprintf("[%s]", stream.MapIndex) + "\n")
			}
			writeStreamFiltered(&b, stream.Stream, stream.TypeIndex, filter, showAll)
		}
	}

	return b.String()
}

// writeStreamFiltered writes the fields of a single Stream to the builder,
// respecting the optional filter and showAll flags.
func writeStreamFiltered(b *strings.Builder, s Stream, typeIdx int, filter map[string]bool, showAll bool) {
	// Index / TypeIndex
	if showAll || filter[FieldIndex] {
		b.WriteString(indent + FieldIndex + separation + strconv.Itoa(s.Index) + "\n")
	}
	if showAll || filter[FieldTypeIndex] {
		b.WriteString(indent + FieldTypeIndex + separation + strconv.Itoa(typeIdx) + "\n")
	}

	// Codec & Profile
	if (showAll || filter[FieldCodec]) && s.CodecName != "" {
		b.WriteString(indent + FieldCodec + separation + s.CodecName + "\n")
	}
	if (showAll || filter[FieldCodecLongName]) && s.CodecLongName != "" {
		b.WriteString(indent + FieldCodecLongName + separation + s.CodecLongName + "\n")
	}
	if (showAll || filter[FieldProfile]) && s.Profile != "" {
		b.WriteString(indent + FieldProfile + separation + s.Profile + "\n")
	}

	// Video
	if (showAll || filter[FieldWidth]) && s.Width > 0 {
		b.WriteString(indent + FieldWidth + separation + strconv.Itoa(s.Width) + "\n")
	}
	if (showAll || filter[FieldHeight]) && s.Height > 0 {
		b.WriteString(indent + FieldHeight + separation + strconv.Itoa(s.Height) + "\n")
	}
	if (showAll || filter[FieldPixFmt]) && s.PixFmt != "" {
		b.WriteString(indent + FieldPixFmt + separation + s.PixFmt + "\n")
	}
	if (showAll || filter[FieldFrameRate]) && s.RFrameRate != "" {
		b.WriteString(indent + FieldFrameRate + separation + s.RFrameRate + "\n")
	}
	if (showAll || filter[FieldAvgFrameRate]) && s.AvgFrameRate != "" {
		b.WriteString(indent + FieldAvgFrameRate + separation + s.AvgFrameRate + "\n")
	}

	// Common
	if (showAll || filter[FieldBitRate]) && s.BitRate != "" {
		b.WriteString(indent + FieldBitRate + separation + s.BitRate + "\n")
	}
	if (showAll || filter[FieldDuration]) && s.Duration != "" {
		b.WriteString(indent + FieldDuration + separation + s.Duration + "\n")
	}
	if (showAll || filter[FieldNbFrames]) && s.NbFrames != "" {
		b.WriteString(indent + FieldNbFrames + separation + s.NbFrames + "\n")
	}

	// Audio
	if (showAll || filter[FieldChannelLayout]) && s.ChannelLayout != "" {
		b.WriteString(indent + FieldChannelLayout + separation + s.ChannelLayout + "\n")
	}
	if (showAll || filter[FieldSampleRate]) && s.SampleRate != "" {
		b.WriteString(indent + FieldSampleRate + separation + s.SampleRate + "\n")
	}
	if (showAll || filter[FieldChannels]) && s.Channels > 0 {
		b.WriteString(indent + FieldChannels + separation + strconv.Itoa(s.Channels) + "\n")
	}

	// Data
	if (showAll || filter[FieldCodecTag]) && s.CodecTagString != "" {
		b.WriteString(indent + FieldCodecTag + separation + s.CodecTagString + "\n")
	}

	if (showAll || filter[FieldCodecType]) && s.CodecType != "" {
		b.WriteString(indent + FieldCodecType + separation + s.CodecType + "\n")
	}
	if (showAll || filter[FieldCodecTimeBase]) && s.CodecTimeBase != "" {
		b.WriteString(indent + FieldCodecTimeBase + separation + s.CodecTimeBase + "\n")
	}
	if (showAll || filter[FieldCodecTagHex]) && s.CodecTag != "" {
		b.WriteString(indent + FieldCodecTagHex + separation + s.CodecTag + "\n")
	}
	if (showAll || filter[FieldCodedWidth]) && s.CodedWidth > 0 {
		b.WriteString(indent + FieldCodedWidth + separation + strconv.Itoa(s.CodedWidth) + "\n")
	}
	if (showAll || filter[FieldCodedHeight]) && s.CodedHeight > 0 {
		b.WriteString(indent + FieldCodedHeight + separation + strconv.Itoa(s.CodedHeight) + "\n")
	}
	if (showAll || filter[FieldClosedCaptions]) && s.ClosedCaptions > 0 {
		b.WriteString(indent + FieldClosedCaptions + separation + strconv.Itoa(s.ClosedCaptions) + "\n")
	}
	if (showAll || filter[FieldHasBFrames]) && s.HasBFrames > 0 {
		b.WriteString(indent + FieldHasBFrames + separation + strconv.Itoa(s.HasBFrames) + "\n")
	}
	if (showAll || filter[FieldSampleAspectRatio]) && s.SampleAspectRatio != "" {
		b.WriteString(indent + FieldSampleAspectRatio + separation + s.SampleAspectRatio + "\n")
	}
	if (showAll || filter[FieldDisplayAspectRatio]) && s.DisplayAspectRatio != "" {
		b.WriteString(indent + FieldDisplayAspectRatio + separation + s.DisplayAspectRatio + "\n")
	}
	if (showAll || filter[FieldLevel]) && s.Level > 0 {
		b.WriteString(indent + FieldLevel + separation + strconv.Itoa(s.Level) + "\n")
	}
	if (showAll || filter[FieldColorRange]) && s.ColorRange != "" {
		b.WriteString(indent + FieldColorRange + separation + s.ColorRange + "\n")
	}
	if (showAll || filter[FieldColorSpace]) && s.ColorSpace != "" {
		b.WriteString(indent + FieldColorSpace + separation + s.ColorSpace + "\n")
	}
	if (showAll || filter[FieldColorTransfer]) && s.ColorTransfer != "" {
		b.WriteString(indent + FieldColorTransfer + separation + s.ColorTransfer + "\n")
	}
	if (showAll || filter[FieldColorPrimaries]) && s.ColorPrimaries != "" {
		b.WriteString(indent + FieldColorPrimaries + separation + s.ColorPrimaries + "\n")
	}
	if (showAll || filter[FieldChromaLocation]) && s.ChromaLocation != "" {
		b.WriteString(indent + FieldChromaLocation + separation + s.ChromaLocation + "\n")
	}
	if (showAll || filter[FieldFieldOrder]) && s.FieldOrder != "" {
		b.WriteString(indent + FieldFieldOrder + separation + s.FieldOrder + "\n")
	}
	if (showAll || filter[FieldRefs]) && s.Refs > 0 {
		b.WriteString(indent + FieldRefs + separation + strconv.Itoa(s.Refs) + "\n")
	}
	if (showAll || filter[FieldIsAvc]) && s.IsAvc != "" {
		b.WriteString(indent + FieldIsAvc + separation + s.IsAvc + "\n")
	}
	if (showAll || filter[FieldNalLengthSize]) && s.NalLengthSize != "" {
		b.WriteString(indent + FieldNalLengthSize + separation + s.NalLengthSize + "\n")
	}
	if (showAll || filter[FieldBitsPerRawSample]) && s.BitsPerRawSample != "" {
		b.WriteString(indent + FieldBitsPerRawSample + separation + s.BitsPerRawSample + "\n")
	}
	if (showAll || filter[FieldSampleFmt]) && s.SampleFmt != "" {
		b.WriteString(indent + FieldSampleFmt + separation + s.SampleFmt + "\n")
	}
	if (showAll || filter[FieldBitsPerSample]) && s.BitsPerSample > 0 {
		b.WriteString(indent + FieldBitsPerSample + separation + strconv.Itoa(s.BitsPerSample) + "\n")
	}
	if (showAll || filter[FieldID]) && s.ID != "" {
		b.WriteString(indent + FieldID + separation + s.ID + "\n")
	}
	if (showAll || filter[FieldTimeBase]) && s.TimeBase != "" {
		b.WriteString(indent + FieldTimeBase + separation + s.TimeBase + "\n")
	}
	if (showAll || filter[FieldStartPts]) && s.StartPts != 0 {
		b.WriteString(indent + FieldStartPts + separation + strconv.FormatInt(s.StartPts, 10) + "\n")
	}
	if (showAll || filter[FieldStartTime]) && s.StartTime != "" {
		b.WriteString(indent + FieldStartTime + separation + s.StartTime + "\n")
	}
	if (showAll || filter[FieldDurationTS]) && s.DurationTS != 0 {
		b.WriteString(indent + FieldDurationTS + separation + strconv.FormatInt(s.DurationTS, 10) + "\n")
	}
	if (showAll || filter[FieldMaxBitRate]) && s.MaxBitRate != "" {
		b.WriteString(indent + FieldMaxBitRate + separation + s.MaxBitRate + "\n")
	}
	if (showAll || filter[FieldNbReadFrames]) && s.NbReadFrames != "" {
		b.WriteString(indent + FieldNbReadFrames + separation + s.NbReadFrames + "\n")
	}
	if (showAll || filter[FieldNbReadPackets]) && s.NbReadPackets != "" {
		b.WriteString(indent + FieldNbReadPackets + separation + s.NbReadPackets + "\n")
	}

	if showAll && s.Disposition != nil {
		b.WriteString(dispsitionToString(s.Disposition))
	}
	if showAll && s.Tags != nil {
		b.WriteString(tagsToString(s.Tags))
	}
}

// tagsToString formats a string→string tag map as an indented block.
func tagsToString(tags map[string]string) string {
	var s strings.Builder
	s.WriteString(indent + FieldTags + separation + "\n")
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if v, ok := tags[k]; ok {
			if v == "" {
				continue
			}
			s.WriteString(indent + indent + k + separation + v + "\n")
		}
	}
	return s.String()
}

// dispsitionToString formats a string→int disposition map as an indented block.
func dispsitionToString(disp map[string]int) string {
	var s strings.Builder
	s.WriteString(indent + FieldDisposition + separation + "\n")
	keys := make([]string, 0, len(disp))
	for k := range disp {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if v, ok := disp[k]; ok {
			if v == 0 {
				continue
			}
			s.WriteString(indent + indent + k + separation + strconv.Itoa(v) + "\n")
		}
	}
	return s.String()
}

// HashFileWithProgress вычисляет SHA-256 хеш файла с прогресс-баром.
// progressInterval – интервал обновления вывода (рекомендуется 5-10 сек для больших файлов).
func HashFileWithProgress(path string, progressInterval time.Duration) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("стат файла: %w", err)
	}
	totalSize := fi.Size()
	if totalSize == 0 {
		hash := sha256.Sum256(nil)
		return hex.EncodeToString(hash[:]), nil
	}

	var bytesRead int64
	done := make(chan struct{})
	defer func() {
		close(done)
		time.Sleep(50 * time.Millisecond)
	}()

	// Прогресс-горутина
	if progressInterval > 0 {
		go func() {
			ticker := time.NewTicker(progressInterval)
			defer ticker.Stop()

			start := time.Now()
			lastBytes := int64(0)
			lastTime := start
			first := true

			for {
				select {
				case <-done:
					read := atomic.LoadInt64(&bytesRead)
					elapsed := time.Since(start)
					percent := float64(read) / float64(totalSize) * 100
					speed := float64(read) / elapsed.Seconds() / 1_000_000
					fmt.Printf("\rПрогресс: %.2f%% (%.2f МБ/с) — завершено за %v\n",
						percent, speed, formatDuration(elapsed))
					return
				case <-ticker.C:
					read := atomic.LoadInt64(&bytesRead)
					now := time.Now()
					percent := float64(read) / float64(totalSize) * 100

					// Пропускаем первый тик, если данных мало
					if first {
						first = false
						lastBytes = read
						lastTime = now
						fmt.Printf("\rПрогресс: %.2f%% (инициализация...)   ", percent)
						continue
					}

					intervalBytes := read - lastBytes
					intervalTime := now.Sub(lastTime).Seconds()
					if intervalBytes > 0 && intervalTime > 0 {
						speed := float64(intervalBytes) / intervalTime / 1_000_000
						remaining := totalSize - read
						eta := time.Duration(float64(remaining) / (float64(intervalBytes) / intervalTime))
						fmt.Printf("\rПрогресс: %.2f%% (%.2f МБ/с), осталось %v   ",
							percent, speed, formatDuration(eta))
					} else {
						fmt.Printf("\rПрогресс: %.2f%% (ожидание данных...)   ", percent)
					}

					lastBytes = read
					lastTime = now
				}
			}
		}()
	}

	// Хеширование с большим буфером (4 МБ)
	hasher := sha256.New()
	buf := make([]byte, 4*1024*1024) // 4 MB
	for {
		n, err := f.Read(buf)
		if n > 0 {
			hasher.Write(buf[:n])
			atomic.AddInt64(&bytesRead, int64(n))
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("ошибка чтения: %w", err)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
