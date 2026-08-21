package format

import (
	"fmt"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// Format представляет видеофильтр format из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#format
type Format struct {
	// PixFmts — список форматов пикселей, разделённых '|', например "yuv420p|yuv444p".
	PixFmts string `json:"pix_fmts,omitempty"`
	// ColorSpaces — список цветовых пространств, разделённых '|', например "bt709|bt470bg".
	ColorSpaces string `json:"color_spaces,omitempty"`
	// ColorRanges — список диапазонов цвета, разделённых '|', например "tv|pc".
	ColorRanges string `json:"color_ranges,omitempty"`
	// AlphaModes — список альфа-режимов, разделённых '|', например "straight|premultiplied".
	AlphaModes string `json:"alpha_modes,omitempty"`

	err error
}

// New создаёт фильтр Format и применяет переданные опции.
// Поддерживаются опции: pix_fmts, color_spaces, color_ranges, alpha_modes.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Format {
	f := &Format{}
	for _, opt := range opts {
		if err := f.apply(opt); err != nil {
			f.err = err
			break
		}
	}
	if f.err == nil {
		f.err = f.Validate()
	}
	return f
}

func (f *Format) apply(opt options.Option) error {
	switch opt.Key {
	case "pix_fmts":
		if opt.Value == "" {
			return fmt.Errorf("format: pix_fmts cannot be empty")
		}
		f.PixFmts = opt.Value
	case "color_spaces":
		if opt.Value == "" {
			return fmt.Errorf("format: color_spaces cannot be empty")
		}
		f.ColorSpaces = opt.Value
	case "color_ranges":
		if opt.Value == "" {
			return fmt.Errorf("format: color_ranges cannot be empty")
		}
		f.ColorRanges = opt.Value
	case "alpha_modes":
		if opt.Value == "" {
			return fmt.Errorf("format: alpha_modes cannot be empty")
		}
		f.AlphaModes = opt.Value
	default:
		return fmt.Errorf("format: unknown option %q", opt.Key)
	}
	return nil
}

// Validate проверяет целостность установленных полей.
// Дополнительных проверок нет.
func (f *Format) Validate() error {
	return nil
}

// String возвращает строку фильтра (например, "format=pix_fmts=yuv420p").
// При ошибке возвращает пустую строку.
func (f *Format) String() string {
	if f.Err() != nil {
		return ""
	}
	var parts []string
	if f.PixFmts != "" {
		parts = append(parts, "pix_fmts="+f.PixFmts)
	}
	if f.ColorSpaces != "" {
		parts = append(parts, "color_spaces="+f.ColorSpaces)
	}
	if f.ColorRanges != "" {
		parts = append(parts, "color_ranges="+f.ColorRanges)
	}
	if f.AlphaModes != "" {
		parts = append(parts, "alpha_modes="+f.AlphaModes)
	}
	if len(parts) == 0 {
		return "format"
	}
	return "format=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (f *Format) Err() error {
	if f.err != nil {
		return f.err
	}
	return f.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -vf.
func (f *Format) ProvideOption() options.Option {
	if f.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: f.String()}
}