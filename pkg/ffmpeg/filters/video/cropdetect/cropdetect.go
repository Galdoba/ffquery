package cropdetect

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
	"github.com/Galdoba/ffcmd/ffmpeg/utils"
)

// Cropdetect представляет видеофильтр cropdetect из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#cropdetect
type Cropdetect struct {
	// Mode определяет метод детекции. Допустимые значения: "black" (по умолчанию) и "mvedges".
	Mode string `json:"mode,omitempty"`
	// Limit — пороговое значение чёрного. Может быть целым от 0 до 255 или дробным от 0.0 до 1.0.
	// По умолчанию 24. Если не задано, используется значение по умолчанию.
	Limit *float64 `json:"limit,omitempty"`
	// Round — значение, которому должны быть кратны ширина и высота. По умолчанию 16.
	// Используйте 2, чтобы получить только чётные размеры.
	Round *int `json:"round,omitempty"`
	// Skip — количество начальных кадров, пропускаемых при оценке. По умолчанию 2. Диапазон от 0 до INT_MAX.
	Skip *int `json:"skip,omitempty"`
	// ResetCount, Reset — счётчик кадров, после которого детектор сбрасывает предыдущую наибольшую область
	// и начинает заново. По умолчанию 0 (никогда не сбрасывать). Полезно при логотипах.
	ResetCount *int `json:"reset_count,omitempty"`
	// MvThreshold — порог движения в пикселях для детекции движения. По умолчанию 8.
	MvThreshold *int `json:"mv_threshold,omitempty"`
	// Low — нижний порог алгоритма Кэнни. Диапазон [0,1], по умолчанию 5/255.
	Low *float64 `json:"low,omitempty"`
	// High — верхний порог алгоритма Кэнни. Диапазон [0,1], по умолчанию 15/255.
	High *float64 `json:"high,omitempty"`

	err error
}

// New создаёт фильтр Cropdetect и применяет переданные опции.
// Поддерживаются опции: mode, limit, round, skip, reset_count/reset, mv_threshold, low, high.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Cropdetect {
	c := &Cropdetect{}
	for _, opt := range opts {
		if err := c.apply(opt); err != nil {
			c.err = err
			break
		}
	}
	if c.err == nil {
		c.err = c.Validate()
	}
	return c
}

// apply валидирует опцию и устанавливает соответствующее поле.
func (c *Cropdetect) apply(opt options.Option) error {
	switch opt.Key {
	case "mode":
		if opt.Value != "black" && opt.Value != "mvedges" {
			return fmt.Errorf("cropdetect: mode must be 'black' or 'mvedges', got %q", opt.Value)
		}
		c.Mode = opt.Value
	case "limit":
		v, err := utils.ParseFloat64Bounded(opt.Value, 0, 255)
		if err != nil {
			return fmt.Errorf("cropdetect: invalid limit value %q: %w", opt.Value, err)
		}
		c.Limit = &v
	case "round":
		n, err := strconv.Atoi(opt.Value)
		if err != nil {
			return fmt.Errorf("cropdetect: round must be an integer, got %q", opt.Value)
		}
		if n <= 0 {
			return fmt.Errorf("cropdetect: round must be positive, got %d", n)
		}
		c.Round = &n
	case "skip":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 0 {
			return fmt.Errorf("cropdetect: skip must be a non-negative integer, got %q", opt.Value)
		}
		c.Skip = &n
	case "reset_count", "reset":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 0 {
			return fmt.Errorf("cropdetect: reset_count must be a non-negative integer, got %q", opt.Value)
		}
		c.ResetCount = &n
	case "mv_threshold":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 0 {
			return fmt.Errorf("cropdetect: mv_threshold must be a non-negative integer, got %q", opt.Value)
		}
		c.MvThreshold = &n
	case "low":
		v, err := utils.ParseFloat64Bounded(opt.Value, 0, 1)
		if err != nil {
			return fmt.Errorf("cropdetect: invalid low value %q: %w", opt.Value, err)
		}
		c.Low = &v
	case "high":
		v, err := utils.ParseFloat64Bounded(opt.Value, 0, 1)
		if err != nil {
			return fmt.Errorf("cropdetect: invalid high value %q: %w", opt.Value, err)
		}
		c.High = &v
	default:
		return fmt.Errorf("cropdetect: unknown option %q", opt.Key)
	}
	return nil
}

// Validate проверяет целостность установленных полей.
// Если заданы оба порога, low должен быть <= high.
func (c *Cropdetect) Validate() error {
	if c.Low != nil && c.High != nil && *c.Low > *c.High {
		return fmt.Errorf("cropdetect: low threshold must be <= high threshold")
	}
	return nil
}

// String возвращает строку фильтра (например, "cropdetect=mode=mvedges:limit=0.1:round=2").
// При ошибке возвращает пустую строку.
func (c *Cropdetect) String() string {
	if c.Err() != nil {
		return ""
	}
	var parts []string

	// Порядок в соответствии с документацией
	if c.Mode != "" {
		parts = append(parts, "mode="+c.Mode)
	}
	if c.Limit != nil {
		parts = append(parts, fmt.Sprintf("limit=%v", *c.Limit))
	}
	if c.Round != nil {
		parts = append(parts, fmt.Sprintf("round=%d", *c.Round))
	}
	if c.Skip != nil {
		parts = append(parts, fmt.Sprintf("skip=%d", *c.Skip))
	}
	if c.ResetCount != nil {
		parts = append(parts, fmt.Sprintf("reset_count=%d", *c.ResetCount))
	}
	if c.MvThreshold != nil {
		parts = append(parts, fmt.Sprintf("mv_threshold=%d", *c.MvThreshold))
	}
	if c.Low != nil {
		parts = append(parts, fmt.Sprintf("low=%v", *c.Low))
	}
	if c.High != nil {
		parts = append(parts, fmt.Sprintf("high=%v", *c.High))
	}

	if len(parts) == 0 {
		return "cropdetect"
	}
	return "cropdetect=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (c *Cropdetect) Err() error {
	if c.err != nil {
		return c.err
	}
	return c.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -vf.
func (c *Cropdetect) ProvideOption() options.Option {
	if c.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: c.String()}
}
