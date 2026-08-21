package scale

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

func TestNewDefault(t *testing.T) {
	s := New()
	if err := s.Err(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := s.String(); got != "scale" {
		t.Errorf("expected 'scale', got %q", got)
	}
}

func TestNewWithWidthHeight(t *testing.T) {
	s := New(
		options.Opt("width", "iw/2"),
		options.Opt("height", "ih"),
	)
	if err := s.Err(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "scale=w=iw/2:h=ih"
	if got := s.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestNewWithSize(t *testing.T) {
	s := New(options.Opt("size", "1280x720"))
	if err := s.Err(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "scale=size=1280x720"
	if got := s.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestNewWithAliases(t *testing.T) {
	s := New(
		options.Opt("w", "200"),
		options.Opt("h", "100"),
	)
	if err := s.Err(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "scale=w=200:h=100"
	if got := s.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}

	s2 := New(options.Opt("s", "vga"))
	if got := s2.String(); got != "scale=size=vga" {
		t.Errorf("expected %q, got %q", "scale=size=vga", got)
	}
}

func TestNewWidthAndSizeConflict(t *testing.T) {
	s := New(
		options.Opt("width", "200"),
		options.Opt("size", "vga"),
	)
	if err := s.Err(); err == nil {
		t.Fatal("expected error, got nil")
	} else if !strings.Contains(err.Error(), "size cannot be used together") {
		t.Errorf("unexpected error message: %v", err)
	}
	if s.String() != "" {
		t.Errorf("expected empty string on error, got %q", s.String())
	}
}

func TestNewUnknownOption(t *testing.T) {
	s := New(options.Opt("foo", "bar"))
	if err := s.Err(); err == nil {
		t.Fatal("expected error, got nil")
	} else if !strings.Contains(err.Error(), "unknown option") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewInvalidEval(t *testing.T) {
	s := New(options.Opt("eval", "bad"))
	if err := s.Err(); err == nil {
		t.Fatal("expected error, got nil")
	} else if !strings.Contains(err.Error(), "eval must be 'init' or 'frame'") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewInvalidInterl(t *testing.T) {
	s := New(options.Opt("interl", "2"))
	if err := s.Err(); err == nil {
		t.Fatal("expected error, got nil")
	} else if !strings.Contains(err.Error(), "interl must be -1, 0, or 1") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewValidInterl(t *testing.T) {
	s := New(options.Opt("interl", "1"))
	if err := s.Err(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "scale=interl=1"
	if got := s.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestNewInvalidParam0(t *testing.T) {
	s := New(options.Opt("param0", "abc"))
	if err := s.Err(); err == nil {
		t.Fatal("expected error, got nil")
	} else if !strings.Contains(err.Error(), "invalid param0 value") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNewValidResetSAR(t *testing.T) {
	s := New(options.Opt("reset_sar", "true"))
	if err := s.Err(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "scale=reset_sar=1"
	if got := s.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestForceDivisibleByRequiresForceOriginalAspectRatio(t *testing.T) {
	s := New(options.Opt("force_divisible_by", "2"))
	if err := s.Err(); err == nil {
		t.Fatal("expected error, got nil")
	} else if !strings.Contains(err.Error(), "force_divisible_by requires force_original_aspect_ratio") {
		t.Errorf("unexpected error message: %v", err)
	}

	s2 := New(
		options.Opt("force_original_aspect_ratio", "decrease"),
		options.Opt("force_divisible_by", "2"),
	)
	if err := s2.Err(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(s2.String(), "force_divisible_by=2") {
		t.Errorf("expected force_divisible_by=2 in string, got %q", s2.String())
	}
}

func TestProvideOption(t *testing.T) {
	s := New(options.Opt("width", "100"))
	opt := s.ProvideOption()
	if opt.Key != "-vf" {
		t.Errorf("expected key '-vf', got %q", opt.Key)
	}
	if opt.Value != "scale=w=100" {
		t.Errorf("expected value 'scale=w=100', got %q", opt.Value)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	original := New(
		options.Opt("width", "iw/2"),
		options.Opt("height", "ih"),
		options.Opt("force_original_aspect_ratio", "decrease"),
		options.Opt("force_divisible_by", "2"),
	)
	if err := original.Err(); err != nil {
		t.Fatalf("original has error: %v", err)
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var loaded Scale
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if err := loaded.Err(); err != nil {
		t.Fatalf("loaded has error: %v", err)
	}
	if loaded.String() != original.String() {
		t.Errorf("expected same string after roundtrip, got %q vs %q", loaded.String(), original.String())
	}
}

func TestManualCreationAndValidation(t *testing.T) {
	s := &Scale{
		Size:  "1280x720",
		Width: "100",
	}
	if err := s.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	} else if !strings.Contains(err.Error(), "size cannot be used together") {
		t.Errorf("unexpected error: %v", err)
	}

	s2 := &Scale{
		Width:  "100",
		Height: "200",
	}
	if err := s2.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}