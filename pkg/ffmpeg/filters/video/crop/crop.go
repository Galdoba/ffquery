package crop

import (
	"fmt"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
	"github.com/Galdoba/ffcmd/ffmpeg/utils"
)

// Crop представляет видеофильтр crop из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#crop
type Crop struct {
	// W, OutW — ширина выходного видео. По умолчанию iw.
	// Выражение вычисляется один раз при конфигурации фильтра или при команде 'w'/'out_w'.
	W string `json:"w,omitempty"`
	// H, OutH — высота выходного видео. По умолчанию ih.
	// Выражение вычисляется один раз при конфигурации фильтра или при команде 'h'/'out_h'.
	H string `json:"h,omitempty"`
	// X — горизонтальная позиция левого края выходного видео во входном.
	// По умолчанию (in_w-out_w)/2. Выражение вычисляется для каждого кадра.
	X string `json:"x,omitempty"`
	// Y — вертикальная позиция верхнего края выходного видео во входном.
	// По умолчанию (in_h-out_h)/2. Выражение вычисляется для каждого кадра.
	Y string `json:"y,omitempty"`
	// KeepAspect — если 1, выходное соотношение сторон будет принудительно сохранено как у входа
	// за счёт изменения Sample Aspect Ratio. По умолчанию 0.
	KeepAspect *bool `json:"keep_aspect,omitempty"`
	// Exact — если true, точное кадрирование без округления для субдискретизированных форматов.
	// По умолчанию false.
	Exact *bool `json:"exact,omitempty"`

	err error
}

// New создаёт фильтр Crop и применяет переданные опции.
// Поддерживаются опции: w/out_w, h/out_h, x, y, keep_aspect, exact.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Crop {
	c := &Crop{}
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
func (c *Crop) apply(opt options.Option) error {
	switch opt.Key {
	case "w", "out_w":
		if opt.Value == "" {
			return fmt.Errorf("crop: width expression cannot be empty")
		}
		c.W = opt.Value
	case "h", "out_h":
		if opt.Value == "" {
			return fmt.Errorf("crop: height expression cannot be empty")
		}
		c.H = opt.Value
	case "x":
		if opt.Value == "" {
			return fmt.Errorf("crop: x expression cannot be empty")
		}
		c.X = opt.Value
	case "y":
		if opt.Value == "" {
			return fmt.Errorf("crop: y expression cannot be empty")
		}
		c.Y = opt.Value
	case "keep_aspect":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("crop: keep_aspect must be boolean, got %q", opt.Value)
		}
		c.KeepAspect = &b
	case "exact":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("crop: exact must be boolean, got %q", opt.Value)
		}
		c.Exact = &b
	default:
		return fmt.Errorf("crop: unknown option %q", opt.Key)
	}
	return nil
}

// Validate проверяет целостность установленных полей.
// Дополнительных проверок нет, так как все значения строковые и не конфликтуют.
func (c *Crop) Validate() error {
	return nil
}

// String возвращает строку фильтра (например, "crop=w=100:h=100:x=12:y=34").
// При ошибке возвращает пустую строку.
func (c *Crop) String() string {
	if c.Err() != nil {
		return ""
	}
	var parts []string

	// Порядок: w, h, x, y, keep_aspect, exact
	if c.W != "" {
		parts = append(parts, "w="+c.W)
	}
	if c.H != "" {
		parts = append(parts, "h="+c.H)
	}
	if c.X != "" {
		parts = append(parts, "x="+c.X)
	}
	if c.Y != "" {
		parts = append(parts, "y="+c.Y)
	}
	if c.KeepAspect != nil {
		if *c.KeepAspect {
			parts = append(parts, "keep_aspect=1")
		} else {
			parts = append(parts, "keep_aspect=0")
		}
	}
	if c.Exact != nil {
		if *c.Exact {
			parts = append(parts, "exact=1")
		} else {
			parts = append(parts, "exact=0")
		}
	}

	if len(parts) == 0 {
		return "crop"
	}
	return "crop=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (c *Crop) Err() error {
	if c.err != nil {
		return c.err
	}
	return c.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -vf.
func (c *Crop) ProvideOption() options.Option {
	if c.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: c.String()}
}
