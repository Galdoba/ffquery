package subtitles

import (
	"fmt"
	"strings"
)

// Константы для Alignment (SSA, сдвиг на основе 4).
const (
	AlignmentBottomLeft   = 1
	AlignmentBottomCenter = 2
	AlignmentBottomRight  = 3
	AlignmentTopLeft      = 5
	AlignmentTopCenter    = 6
	AlignmentTopRight     = 7
	AlignmentMiddleLeft   = 9
	AlignmentMiddleCenter = 10
	AlignmentMiddleRight  = 11
)

// Константы для BorderStyle.
const (
	BorderStyleOutline = 1 // контур
	BorderStyleBox     = 3 // прямоугольная подложка
)

// ASSStyle представляет параметры стиля субтитров в формате ASS/SSA.
// Нулевые значения полей (пустые строки, 0, false) игнорируются при формировании force_style.
type ASSStyle struct {
	Fontname        string  `json:"fontname,omitempty"`
	Fontsize        float64 `json:"fontsize,omitempty"`
	PrimaryColour   string  `json:"primary_colour,omitempty"`
	SecondaryColour string  `json:"secondary_colour,omitempty"`
	OutlineColour   string  `json:"outline_colour,omitempty"`
	BackColour      string  `json:"back_colour,omitempty"`
	Bold            bool    `json:"bold,omitempty"`
	Italic          bool    `json:"italic,omitempty"`
	Underline       bool    `json:"underline,omitempty"`
	StrikeOut       bool    `json:"strikeout,omitempty"`
	ScaleX          float64 `json:"scale_x,omitempty"`
	ScaleY          float64 `json:"scale_y,omitempty"`
	Spacing         float64 `json:"spacing,omitempty"`
	Angle           float64 `json:"angle,omitempty"`
	BorderStyle     int     `json:"border_style,omitempty"`
	Outline         float64 `json:"outline,omitempty"`
	Shadow          float64 `json:"shadow,omitempty"`
	Alignment       int     `json:"alignment,omitempty"`
	MarginL         int     `json:"margin_l,omitempty"`
	MarginR         int     `json:"margin_r,omitempty"`
	MarginV         int     `json:"margin_v,omitempty"`
	Encoding        int     `json:"encoding,omitempty"`
}

// DefaultASSStyle возвращает стиль с параметрами, использованными в шаблоне хардсаба.
func DefaultASSStyle() *ASSStyle {
	return &ASSStyle{
		Fontname:        "Segoe UI",
		Fontsize:        19.4,
		PrimaryColour:   "&H00EBEBEB",
		SecondaryColour: "&H000000FF",
		OutlineColour:   "&H00000000",
		BackColour:      "&H80000000",
		Bold:            false,
		Italic:          false,
		Underline:       false,
		StrikeOut:       false,
		ScaleX:          1,
		ScaleY:          1,
		Spacing:         0,
		Angle:           0,
		BorderStyle:     BorderStyleOutline,
		Outline:         0.5,
		Shadow:          0,
		Alignment:       AlignmentBottomCenter, // 2, как в хардсабе
		MarginL:         5,
		MarginR:         5,
		MarginV:         40,
		Encoding:        1,
	}
}

// String возвращает строку для опции force_style.
// Включаются только поля с ненулевыми значениями.
func (s *ASSStyle) String() string {
	var parts []string

	if s.Fontname != "" {
		parts = append(parts, "Fontname="+s.Fontname)
	}
	if s.Fontsize != 0 {
		parts = append(parts, fmt.Sprintf("Fontsize=%g", s.Fontsize))
	}
	if s.PrimaryColour != "" {
		parts = append(parts, "PrimaryColour="+s.PrimaryColour)
	}
	if s.SecondaryColour != "" {
		parts = append(parts, "SecondaryColour="+s.SecondaryColour)
	}
	if s.OutlineColour != "" {
		parts = append(parts, "OutlineColour="+s.OutlineColour)
	}
	if s.BackColour != "" {
		parts = append(parts, "BackColour="+s.BackColour)
	}
	if s.Bold {
		parts = append(parts, "Bold=1")
	}
	if s.Italic {
		parts = append(parts, "Italic=1")
	}
	if s.Underline {
		parts = append(parts, "Underline=1")
	}
	if s.StrikeOut {
		parts = append(parts, "StrikeOut=1")
	}
	if s.ScaleX != 0 {
		parts = append(parts, fmt.Sprintf("ScaleX=%g", s.ScaleX))
	}
	if s.ScaleY != 0 {
		parts = append(parts, fmt.Sprintf("ScaleY=%g", s.ScaleY))
	}
	if s.Spacing != 0 {
		parts = append(parts, fmt.Sprintf("Spacing=%g", s.Spacing))
	}
	if s.Angle != 0 {
		parts = append(parts, fmt.Sprintf("Angle=%g", s.Angle))
	}
	if s.BorderStyle != 0 {
		parts = append(parts, fmt.Sprintf("BorderStyle=%d", s.BorderStyle))
	}
	if s.Outline != 0 {
		parts = append(parts, fmt.Sprintf("Outline=%g", s.Outline))
	}
	if s.Shadow != 0 {
		parts = append(parts, fmt.Sprintf("Shadow=%g", s.Shadow))
	}
	if s.Alignment != 0 {
		parts = append(parts, fmt.Sprintf("Alignment=%d", s.Alignment))
	}
	if s.MarginL != 0 {
		parts = append(parts, fmt.Sprintf("MarginL=%d", s.MarginL))
	}
	if s.MarginR != 0 {
		parts = append(parts, fmt.Sprintf("MarginR=%d", s.MarginR))
	}
	if s.MarginV != 0 {
		parts = append(parts, fmt.Sprintf("MarginV=%d", s.MarginV))
	}
	if s.Encoding != 0 {
		parts = append(parts, fmt.Sprintf("Encoding=%d", s.Encoding))
	}

	return strings.Join(parts, ",")
}