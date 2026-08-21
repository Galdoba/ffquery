// Package metricfile provides binary read/write for Y-block metric files.
package metricfile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	// Magic bytes to identify the format.
	Magic = "YBLK"

	// Current version of the format.
	Version uint8 = 1

	// HeaderSize is the fixed header length in bytes.
	HeaderSize = 64
)

var (
	ErrInvalidMagic    = errors.New("invalid magic bytes")
	ErrVersion         = errors.New("unsupported version")
	ErrShortHeader     = errors.New("short header read")
	ErrFrameOutOfRange = errors.New("frame index out of range")
)

// Header contains all metadata stored at the beginning of the file.
type Header struct {
	Magic       [4]byte // "YBLK"
	Version     uint8
	Flags       uint8
	Cols        uint16
	Rows        uint16
	VideoWidth  uint32
	VideoHeight uint32
	FrameCount  uint64
	FPSNum      uint32
	FPSDen      uint32
	Reserved    [30]byte
}

// NewHeader creates a Header with default magic and version, zeroed reserved.
func NewHeader(cols, rows int, videoW, videoH int, fpsNum, fpsDen int) Header {
	h := Header{
		Version:     Version,
		Cols:        uint16(cols),
		Rows:        uint16(rows),
		VideoWidth:  uint32(videoW),
		VideoHeight: uint32(videoH),
		FPSNum:      uint32(fpsNum),
		FPSDen:      uint32(fpsDen),
	}
	copy(h.Magic[:], Magic)
	return h
}

// FrameSize returns the size of one frame vector in bytes.
func (h Header) FrameSize() int {
	return int(h.Cols) * int(h.Rows)
}

// FrameCountByFileSize computes the total number of frames based on file size.
func (h Header) FrameCountByFileSize(fileSize int64) uint64 {
	dataSize := fileSize - int64(HeaderSize)
	if dataSize < 0 {
		return 0
	}
	return uint64(dataSize / int64(h.FrameSize()))
}

// WriteTo serialises the header and writes it to w.
func (h Header) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, HeaderSize)
	copy(buf[0:4], h.Magic[:])
	buf[4] = h.Version
	buf[5] = h.Flags
	binary.LittleEndian.PutUint16(buf[6:8], h.Cols)
	binary.LittleEndian.PutUint16(buf[8:10], h.Rows)
	binary.LittleEndian.PutUint32(buf[10:14], h.VideoWidth)
	binary.LittleEndian.PutUint32(buf[14:18], h.VideoHeight)
	binary.LittleEndian.PutUint64(buf[18:26], h.FrameCount)
	binary.LittleEndian.PutUint32(buf[26:30], h.FPSNum)
	binary.LittleEndian.PutUint32(buf[30:34], h.FPSDen)
	// reserved bytes already zero
	n, err := w.Write(buf)
	return int64(n), err
}

// ReadHeader reads and parses a Header from r.
func ReadHeader(r io.Reader) (Header, error) {
	var h Header
	buf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return h, fmt.Errorf("%w: %v", ErrShortHeader, err)
	}
	copy(h.Magic[:], buf[0:4])
	if string(h.Magic[:]) != Magic {
		return h, ErrInvalidMagic
	}
	h.Version = buf[4]
	if h.Version != Version {
		return h, fmt.Errorf("%w: got %d, expected %d", ErrVersion, h.Version, Version)
	}
	h.Flags = buf[5]
	h.Cols = binary.LittleEndian.Uint16(buf[6:8])
	h.Rows = binary.LittleEndian.Uint16(buf[8:10])
	h.VideoWidth = binary.LittleEndian.Uint32(buf[10:14])
	h.VideoHeight = binary.LittleEndian.Uint32(buf[14:18])
	h.FrameCount = binary.LittleEndian.Uint64(buf[18:26])
	h.FPSNum = binary.LittleEndian.Uint32(buf[26:30])
	h.FPSDen = binary.LittleEndian.Uint32(buf[30:34])
	// copy reserved bytes if needed later
	copy(h.Reserved[:], buf[34:64])
	return h, nil
}

// Writer provides streaming write of frames after a header.
type Writer struct {
	w   io.Writer
	hdr Header
	buf []byte // buffer for one frame
}

// NewWriter writes the header and returns a Writer ready to accept frames.
func NewWriter(w io.Writer, hdr Header) (*Writer, error) {
	if _, err := hdr.WriteTo(w); err != nil {
		return nil, err
	}
	return &Writer{
		w:   w,
		hdr: hdr,
		buf: make([]byte, hdr.FrameSize()),
	}, nil
}

// WriteFrame writes a single frame vector. The slice length must equal FrameSize().
func (w *Writer) WriteFrame(vector []byte) error {
	if len(vector) != w.hdr.FrameSize() {
		return fmt.Errorf("invalid vector length: got %d, want %d", len(vector), w.hdr.FrameSize())
	}
	// Write directly without buffering.
	_, err := w.Write(vector)
	return err
}

// Write implements io.Writer, allowing the Writer to be used as a sink for raw frames.
func (w *Writer) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

// Reader allows reading frames sequentially or by absolute index.
type Reader struct {
	r    io.ReaderAt // must support random access (e.g., *os.File)
	hdr  Header
	base int64 // offset of first frame data
}

// NewReader reads the header from r and returns a Reader for frame access.
func NewReader(r io.ReaderAt) (*Reader, error) {
	// Read header from the beginning.
	headerBytes := make([]byte, HeaderSize)
	n, err := r.ReadAt(headerBytes, 0)
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if n != HeaderSize {
		return nil, ErrShortHeader
	}
	hdr, err := ReadHeader(&sliceReader{
		b: headerBytes,
		i: 0,
	})
	if err != nil {
		return nil, err
	}
	return &Reader{
		r:    r,
		hdr:  hdr,
		base: int64(HeaderSize),
	}, nil
}

// Header returns the file header.
func (r *Reader) Header() Header {
	return r.hdr
}

// ReadFrame reads the frame with the given zero-based index.
func (r *Reader) ReadFrame(index uint64) ([]byte, error) {
	frameSize := int64(r.hdr.FrameSize())
	offset := r.base + int64(index)*frameSize
	buf := make([]byte, frameSize)
	n, err := r.r.ReadAt(buf, offset)
	if err != nil {
		return nil, err
	}
	if n != int(frameSize) {
		return nil, ErrFrameOutOfRange
	}
	return buf, nil
}

// sliceReader is a helper to turn a byte slice into an io.Reader for header parsing.
type sliceReader struct {
	b []byte
	i int
}

func (s *sliceReader) Read(p []byte) (int, error) {
	if s.i >= len(s.b) {
		return 0, io.EOF
	}
	n := copy(p, s.b[s.i:])
	s.i += n
	return n, nil
}
