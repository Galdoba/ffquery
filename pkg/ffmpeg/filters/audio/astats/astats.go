package astats

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
	"github.com/Galdoba/ffcmd/ffmpeg/utils"
)

// Astats представляет аудиофильтр astats из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#astats
type Astats struct {
	// Length — длина короткого окна в секундах для пикового и RMS измерения.
	// Диапазон: [0 – 10]. По умолчанию 0.05 (50 мс).
	Length *float64 `json:"length,omitempty"`
	// Metadata — если true, включает инъекцию метаданных с ключами lavfi.astats.X.
	// По умолчанию false (выключено).
	Metadata *bool `json:"metadata,omitempty"`
	// Reset — количество кадров, после которого накопленная статистика сбрасывается.
	// 0 (или не задано) означает «никогда не сбрасывать». По умолчанию выключено.
	Reset *int `json:"reset,omitempty"`
	// MeasurePerchannel — какие параметры измерять для каждого канала.
	// Значение: "none", "all" или список ключей через "+", например "Peak_level+RMS_level".
	// По умолчанию "all".
	MeasurePerchannel string `json:"measure_perchannel,omitempty"`
	// MeasureOverall — какие параметры измерять в целом.
	// Значение: "none", "all" или список ключей через "+", например "RMS_level+Peak_count".
	// По умолчанию "all".
	MeasureOverall string `json:"measure_overall,omitempty"`

	err error
}

// Допустимые ключи измерений для per-channel (согласно документации).
var allowedPerChannel = map[string]bool{
	"Bit_depth":           true,
	"Crest_factor":        true,
	"DC_offset":           true,
	"Dynamic_range":       true,
	"Entropy":             true,
	"Flat_factor":         true,
	"Max_difference":      true,
	"Max_level":           true,
	"Mean_difference":     true,
	"Min_difference":      true,
	"Min_level":           true,
	"Noise_floor":         true,
	"Noise_floor_count":   true,
	"Number_of_Infs":      true,
	"Number_of_NaNs":      true,
	"Number_of_denormals": true,
	"Peak_count":          true,
	"Abs_Peak_count":      true,
	"Peak_level":          true,
	"RMS_difference":      true,
	"RMS_peak":            true,
	"RMS_trough":          true,
	"Zero_crossings":      true,
	"Zero_crossings_rate": true,
}

// Допустимые ключи измерений для overall (согласно документации).
var allowedOverall = map[string]bool{
	"Bit_depth":           true,
	"DC_offset":           true,
	"Entropy":             true,
	"Flat_factor":         true,
	"Max_difference":      true,
	"Max_level":           true,
	"Mean_difference":     true,
	"Min_difference":      true,
	"Min_level":           true,
	"Noise_floor":         true,
	"Noise_floor_count":   true,
	"Number_of_Infs":      true,
	"Number_of_NaNs":      true,
	"Number_of_denormals": true,
	"Number_of_samples":   true,
	"Peak_count":          true,
	"Abs_Peak_count":      true,
	"Peak_level":          true,
	"RMS_difference":      true,
	"RMS_level":           true,
	"RMS_peak":            true,
	"RMS_trough":          true,
	"Zero_crossings":      true,
	"Zero_crossings_rate": true,
}

// New создаёт фильтр Astats и применяет переданные опции.
// Поддерживаются опции: length, metadata, reset, measure_perchannel, measure_overall.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Astats {
	a := &Astats{}
	for _, opt := range opts {
		if err := a.apply(opt); err != nil {
			a.err = err
			break
		}
	}
	if a.err == nil {
		a.err = a.Validate()
	}
	return a
}

func (a *Astats) apply(opt options.Option) error {
	switch opt.Key {
	case "length":
		v, err := strconv.ParseFloat(opt.Value, 64)
		if err != nil {
			return fmt.Errorf("astats: invalid length value %q", opt.Value)
		}
		if v < 0 || v > 10 {
			return fmt.Errorf("astats: length must be in range [0, 10], got %v", v)
		}
		a.Length = &v
	case "metadata":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("astats: metadata must be boolean, got %q", opt.Value)
		}
		a.Metadata = &b
	case "reset":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 0 {
			return fmt.Errorf("astats: reset must be a non-negative integer, got %q", opt.Value)
		}
		a.Reset = &n
	case "measure_perchannel":
		if err := validateMeasures(opt.Value, allowedPerChannel); err != nil {
			return fmt.Errorf("astats: %v", err)
		}
		a.MeasurePerchannel = opt.Value
	case "measure_overall":
		if err := validateMeasures(opt.Value, allowedOverall); err != nil {
			return fmt.Errorf("astats: %v", err)
		}
		a.MeasureOverall = opt.Value
	default:
		return fmt.Errorf("astats: unknown option %q", opt.Key)
	}
	return nil
}

// validateMeasures проверяет строку мер.
func validateMeasures(value string, allowed map[string]bool) error {
	if value == "" {
		return fmt.Errorf("measure value cannot be empty")
	}
	if value == "none" || value == "all" {
		return nil
	}
	keys := strings.Split(value, "+")
	for _, k := range keys {
		if !allowed[k] {
			return fmt.Errorf("measure '%s' is not valid", k)
		}
	}
	return nil
}

func (a *Astats) Validate() error {
	// Дополнительных проверок нет, так как всё уже проверено в apply.
	return nil
}

func (a *Astats) String() string {
	if a.Err() != nil {
		return ""
	}
	var parts []string
	if a.Length != nil {
		parts = append(parts, fmt.Sprintf("length=%v", *a.Length))
	}
	if a.Metadata != nil {
		if *a.Metadata {
			parts = append(parts, "metadata=1")
		} else {
			parts = append(parts, "metadata=0")
		}
	}
	if a.Reset != nil {
		parts = append(parts, fmt.Sprintf("reset=%d", *a.Reset))
	}
	if a.MeasurePerchannel != "" {
		parts = append(parts, "measure_perchannel="+a.MeasurePerchannel)
	}
	if a.MeasureOverall != "" {
		parts = append(parts, "measure_overall="+a.MeasureOverall)
	}
	if len(parts) == 0 {
		return "astats"
	}
	return "astats=" + strings.Join(parts, ":")
}

func (a *Astats) Err() error {
	if a.err != nil {
		return a.err
	}
	return a.Validate()
}

func (a *Astats) ProvideOption() options.Option {
	if a.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-af", Value: a.String()}
}
