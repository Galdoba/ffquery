package ffprobe

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const tagLanguage = "language"

// ---------------------------------------------------------------------------
// Core metadata accessors
// ---------------------------------------------------------------------------

func (s Stream) FPS() float64 {
	return parseFraction(s.RFrameRate)
}

func (s Stream) Bitrate() float64 {
	if s.BitRate == "" {
		return 0
	}
	bits, err := strconv.ParseFloat(s.BitRate, 64)
	if err != nil {
		return 0
	}
	return bits / 1000.0
}

func (s Stream) Size() string {
	if s.Width == 0 || s.Height == 0 {
		return ""
	}
	return fmt.Sprintf("%dx%d", s.Width, s.Height)
}

func (s Stream) Language() string {
	return s.Tags[tagLanguage]
}

func (s Stream) Type() string {
	return s.CodecType
}

func (s Stream) Codec() string {
	return s.CodecName
}

// ---------------------------------------------------------------------------
// Duration helpers
// ---------------------------------------------------------------------------

func (s Stream) DurationSeconds() float64 {
	if s.Duration != "" {
		if d, err := strconv.ParseFloat(s.Duration, 64); err == nil {
			return d
		}
	}
	if s.DurationTS != 0 && s.TimeBase != "" {
		if tb := parseFraction(s.TimeBase); tb > 0 {
			return float64(s.DurationTS) * tb
		}
	}
	return 0
}

func (s Stream) DurationTimestamp() string {
	h, m, sec, ms := s.DurationValues()
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, sec, ms)
}

func (s Stream) DurationValues() (int, int, int, int) {
	sec := s.DurationSeconds()
	if sec <= 0 {
		return 0, 0, 0, 0
	}
	hours := int(sec) / 3600
	minutes := (int(sec) % 3600) / 60
	secs := int(sec) % 60
	frac := sec - math.Floor(sec)
	millis := int(math.Round(frac * 1000))
	if millis == 1000 {
		millis = 0
		secs++
		if secs == 60 {
			secs = 0
			minutes++
			if minutes == 60 {
				minutes = 0
				hours++
			}
		}
	}
	return hours, minutes, secs, millis
}

// ---------------------------------------------------------------------------
// Type checks
// ---------------------------------------------------------------------------

func (s Stream) IsVideo() bool    { return s.CodecType == StreamTypeVideo }
func (s Stream) IsAudio() bool    { return s.CodecType == StreamTypeAudio }
func (s Stream) IsSubtitle() bool { return s.CodecType == StreamTypeSubtitle }
func (s Stream) IsData() bool     { return s.CodecType == StreamTypeData }

// ---------------------------------------------------------------------------
// Tags and disposition
// ---------------------------------------------------------------------------

func (s Stream) HasTag(key string) bool {
	v, ok := s.Tags[key]
	return ok && v != ""
}

func (s Stream) GetTag(key string) string {
	if s.Tags == nil {
		return ""
	}
	return s.Tags[key]
}

func (s Stream) DispositionDefault() bool { return s.Disposition["default"] > 0 }
func (s Stream) DispositionForced() bool  { return s.Disposition["forced"] > 0 }

// ---------------------------------------------------------------------------
// Aspect ratio logic
// ---------------------------------------------------------------------------

// SAR returns the sample (pixel) aspect ratio as (width, height) parts.
// e.g. "8:9" returns (8.0, 9.0). Empty string yields (0,0).
func (s Stream) SAR() (float64, float64) {
	if s.SampleAspectRatio == "" {
		return 0, 0
	}
	w, h := parseRatio(s.SampleAspectRatio)
	return w, h
}

// DAR returns the display aspect ratio as (width, height) parts.
// It prefers the explicit DisplayAspectRatio field, otherwise calculates it
// from the resolution and sample aspect ratio.
// e.g. "16:9" returns (16.0, 9.0).
func (s Stream) DAR() (float64, float64) {
	if s.DisplayAspectRatio != "" {
		return parseRatio(s.DisplayAspectRatio)
	}
	// Fallback: compute from width, height, and SAR
	if s.Width > 0 && s.Height > 0 {
		sarW, sarH := s.SAR()
		if sarW == 0 && sarH == 0 {
			// Assume square pixels
			return float64(s.Width), float64(s.Height)
		}
		darW := float64(s.Width) * sarW
		darH := float64(s.Height) * sarH
		// Simplify to reasonable integer pair
		g := int64(gcd(int(darW), int(darH)))
		return darW / float64(g), darH / float64(g)
	}
	return 0, 0
}

// AspectRatio returns a string representation like "16:9" (uses DAR if available,
// otherwise falls back to a named approximation based on plain width/height).
func (s Stream) AspectRatio() string {
	if s.DisplayAspectRatio != "" {
		return s.DisplayAspectRatio
	}
	if s.Width == 0 || s.Height == 0 {
		return ""
	}
	return approximateRatio(s.Width, s.Height)
}

// ---------------------------------------------------------------------------
// Technical parameters
// ---------------------------------------------------------------------------

func (s Stream) SampleRateHz() int {
	if s.SampleRate == "" {
		return 0
	}
	v, _ := strconv.Atoi(s.SampleRate)
	return v
}

func (s Stream) PixelFormat() string {
	return s.PixFmt
}

func (s Stream) Frames() int {
	if s.NbFrames == "" {
		return 0
	}
	// Some codecs report fractional frames? It's an integer anyway.
	n, _ := strconv.Atoi(s.NbFrames)
	return n
}

// SizeFormat returns a human‑readable label for the video resolution
// (e.g. "HD", "UHD", "SD PAL").
func (s Stream) SizeFormat() string {
	w, h := s.Width, s.Height
	if w == 0 || h == 0 {
		return ""
	}
	switch {
	case w == 7680 && h == 4320:
		return "8K"
	case w >= 7680 && h >= 4320:
		return "8K ready"
	case w == 3840 && h == 2160:
		return "UHD"
	case w >= 3840 && h >= 2160:
		return "UHD ready"
	case w == 2560 && h == 1440:
		return "QHD"
	case w >= 2560 && h >= 1440:
		return "QHD ready"
	case w == 1920 && h == 1080:
		return "Full HD"
	case w >= 1920 && h >= 1080:
		return "Full HD ready"
	case w >= 1280 && h >= 720:
		return "HD Ready"
	case w >= 720 && h >= 576:
		return "SD PAL"
	case w >= 720 && h >= 480:
		return "SD NTSC"
	default:
		return fmt.Sprintf("%dx%d", w, h)
	}
}

// ---------------------------------------------------------------------------
// Internal parsing helpers (unchanged from previous, kept for completeness)
// ---------------------------------------------------------------------------

func parseFraction(s string) float64 {
	if s == "" {
		return 0
	}
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 2 {
		num, err1 := strconv.ParseFloat(parts[0], 64)
		den, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && den != 0 {
			return num / den
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f
	}
	return 0
}

func parseRatio(s string) (float64, float64) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	w, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	h, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return w, h
}

func approximateRatio(w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	r := float64(w) / float64(h)
	switch {
	case math.Abs(r-16.0/9.0) < 0.02:
		return "16:9"
	case math.Abs(r-4.0/3.0) < 0.02:
		return "4:3"
	case math.Abs(r-1.0) < 0.02:
		return "1:1"
	case math.Abs(r-2.35) < 0.05:
		return "2.35:1"
	case math.Abs(r-1.85) < 0.05:
		return "1.85:1"
	default:
		g := gcd(w, h)
		return fmt.Sprintf("%d:%d", w/g, h/g)
	}
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
