package subtitles

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
	"github.com/Galdoba/ffcmd/ffmpeg/utils"
)

// Subtitles представляет видеофильтр subtitles из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#subtitles
type Subtitles struct {
	// Filename, F — путь к файлу субтитров. Обязательный параметр.
	Filename string `json:"filename,omitempty"`
	// OriginalSize — размер оригинального видео для корректного масштабирования шрифтов.
	OriginalSize string `json:"original_size,omitempty"`
	// Fontsdir — путь к директории со шрифтами.
	Fontsdir string `json:"fontsdir,omitempty"`
	// Alpha — обрабатывать ли альфа-канал. По умолчанию false.
	Alpha *bool `json:"alpha,omitempty"`
	// Charenc — кодировка символов входного файла субтитров (если не UTF-8).
	Charenc string `json:"charenc,omitempty"`
	// StreamIndex, Si — индекс потока субтитров в контейнере.
	StreamIndex *int `json:"stream_index,omitempty"`
	// ForceStyle — переопределение стиля ASS (строка KEY=VALUE, разделённые запятыми).
	ForceStyle string `json:"force_style,omitempty"`
	// WrapUnicode — использовать ли Unicode Line Breaking Algorithm. По умолчанию true (для не-ASS).
	WrapUnicode *bool `json:"wrap_unicode,omitempty"`
	// Shaping — движок шейпинга: "auto", "simple", "complex". По умолчанию "auto".
	Shaping string `json:"shaping,omitempty"`

	err error
}

// New создаёт фильтр Subtitles и применяет переданные опции.
// Поддерживаются опции: filename/f, original_size, fontsdir, alpha, charenc,
// stream_index/si, force_style, wrap_unicode, shaping.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Subtitles {
	s := &Subtitles{}
	for _, opt := range opts {
		if err := s.apply(opt); err != nil {
			s.err = err
			break
		}
	}
	if s.err == nil {
		s.err = s.Validate()
	}
	return s
}

func (s *Subtitles) apply(opt options.Option) error {
	switch opt.Key {
	case "filename", "f":
		if opt.Value == "" {
			return fmt.Errorf("subtitles: filename cannot be empty")
		}
		s.Filename = opt.Value
	case "original_size":
		if opt.Value == "" {
			return fmt.Errorf("subtitles: original_size cannot be empty")
		}
		s.OriginalSize = opt.Value
	case "fontsdir":
		if opt.Value == "" {
			return fmt.Errorf("subtitles: fontsdir cannot be empty")
		}
		s.Fontsdir = opt.Value
	case "alpha":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("subtitles: alpha must be boolean, got %q", opt.Value)
		}
		s.Alpha = &b
	case "charenc":
		s.Charenc = opt.Value
	case "stream_index", "si":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 0 {
			return fmt.Errorf("subtitles: stream_index must be a non-negative integer, got %q", opt.Value)
		}
		s.StreamIndex = &n
	case "force_style":
		if opt.Value == "" {
			return fmt.Errorf("subtitles: force_style cannot be empty")
		}
		s.ForceStyle = opt.Value
	case "wrap_unicode":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("subtitles: wrap_unicode must be boolean, got %q", opt.Value)
		}
		s.WrapUnicode = &b
	case "shaping":
		if opt.Value != "auto" && opt.Value != "simple" && opt.Value != "complex" {
			return fmt.Errorf("subtitles: shaping must be 'auto', 'simple' or 'complex', got %q", opt.Value)
		}
		s.Shaping = opt.Value
	default:
		return fmt.Errorf("subtitles: unknown option %q", opt.Key)
	}
	return nil
}

func (s *Subtitles) Validate() error {
	if s.Filename == "" {
		return fmt.Errorf("subtitles: filename is required")
	}
	return nil
}

func (s *Subtitles) String() string {
	if s.Err() != nil {
		return ""
	}
	var parts []string
	// Первым выводим filename (как позиционный или с ключом)
	if s.Filename != "" {
		parts = append(parts, "filename="+s.Filename)
	}
	if s.OriginalSize != "" {
		parts = append(parts, "original_size="+s.OriginalSize)
	}
	if s.Fontsdir != "" {
		parts = append(parts, "fontsdir="+s.Fontsdir)
	}
	if s.Alpha != nil {
		if *s.Alpha {
			parts = append(parts, "alpha=1")
		} else {
			parts = append(parts, "alpha=0")
		}
	}
	if s.Charenc != "" {
		parts = append(parts, "charenc="+s.Charenc)
	}
	if s.StreamIndex != nil {
		parts = append(parts, fmt.Sprintf("si=%d", *s.StreamIndex))
	}
	if s.ForceStyle != "" {
		parts = append(parts, "force_style="+s.ForceStyle)
	}
	if s.WrapUnicode != nil {
		if *s.WrapUnicode {
			parts = append(parts, "wrap_unicode=1")
		} else {
			parts = append(parts, "wrap_unicode=0")
		}
	}
	if s.Shaping != "" {
		parts = append(parts, "shaping="+s.Shaping)
	}
	if len(parts) == 0 {
		return "subtitles"
	}
	return "subtitles=" + strings.Join(parts, ":")
}

func (s *Subtitles) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.Validate()
}

func (s *Subtitles) ProvideOption() options.Option {
	if s.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: s.String()}
}
