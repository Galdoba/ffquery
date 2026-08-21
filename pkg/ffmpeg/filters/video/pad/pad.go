package pad

import (
	"fmt"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// Pad представляет видеофильтр pad из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#pad
type Pad struct {
	// Width, W — выражение для ширины выходного изображения (с учётом полей).
	// Если 0, используется ширина входа. По умолчанию 0.
	// Выражение может ссылаться на Height.
	Width string `json:"width,omitempty"`
	// Height, H — выражение для высоты выходного изображения (с учётом полей).
	// Если 0, используется высота входа. По умолчанию 0.
	// Выражение может ссылаться на Width.
	Height string `json:"height,omitempty"`
	// X — смещение входного изображения по горизонтали от левого края.
	// По умолчанию 0. Если отрицательное, входное изображение будет отцентровано.
	X string `json:"x,omitempty"`
	// Y — смещение входного изображения по вертикали от верхнего края.
	// По умолчанию 0. Если отрицательное, входное изображение будет отцентровано.
	Y string `json:"y,omitempty"`
	// Color — цвет заполнения. По умолчанию "black".
	// Синтаксис цвета описан в ffmpeg-utils.
	Color string `json:"color,omitempty"`
	// Eval — когда вычислять выражения width, height, x и y.
	// Значения: "init" (по умолчанию), "frame".
	Eval string `json:"eval,omitempty"`
	// Aspect — если задано, pad будет подгонять под соотношение сторон вместо разрешения.
	// Значение может быть выражением (например, "16/9").
	Aspect string `json:"aspect,omitempty"`

	err error
}

// New создаёт фильтр Pad и применяет переданные опции.
// Поддерживаются опции: width/w, height/h, x, y, color, eval, aspect.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Pad {
	p := &Pad{}
	for _, opt := range opts {
		if err := p.apply(opt); err != nil {
			p.err = err
			break
		}
	}
	if p.err == nil {
		p.err = p.Validate()
	}
	return p
}

// apply валидирует опцию и устанавливает соответствующее поле.
func (p *Pad) apply(opt options.Option) error {
	switch opt.Key {
	case "width", "w":
		if opt.Value == "" {
			return fmt.Errorf("pad: width expression cannot be empty")
		}
		p.Width = opt.Value
	case "height", "h":
		if opt.Value == "" {
			return fmt.Errorf("pad: height expression cannot be empty")
		}
		p.Height = opt.Value
	case "x":
		if opt.Value == "" {
			return fmt.Errorf("pad: x expression cannot be empty")
		}
		p.X = opt.Value
	case "y":
		if opt.Value == "" {
			return fmt.Errorf("pad: y expression cannot be empty")
		}
		p.Y = opt.Value
	case "color":
		if opt.Value == "" {
			return fmt.Errorf("pad: color cannot be empty")
		}
		p.Color = opt.Value
	case "eval":
		if opt.Value != "init" && opt.Value != "frame" {
			return fmt.Errorf("pad: eval must be 'init' or 'frame', got %q", opt.Value)
		}
		p.Eval = opt.Value
	case "aspect":
		if opt.Value == "" {
			return fmt.Errorf("pad: aspect cannot be empty")
		}
		p.Aspect = opt.Value
	default:
		return fmt.Errorf("pad: unknown option %q", opt.Key)
	}
	return nil
}

// Validate проверяет целостность установленных полей.
// Дополнительных проверок нет, так как все значения строковые и не конфликтуют.
func (p *Pad) Validate() error {
	return nil
}

// String возвращает строку фильтра (например, "pad=width=640:height=480:x=0:y=40:color=violet").
// При ошибке возвращает пустую строку.
func (p *Pad) String() string {
	if p.Err() != nil {
		return ""
	}
	var parts []string

	// Порядок: width, height, x, y, color, eval, aspect
	if p.Width != "" {
		parts = append(parts, "width="+p.Width)
	}
	if p.Height != "" {
		parts = append(parts, "height="+p.Height)
	}
	if p.X != "" {
		parts = append(parts, "x="+p.X)
	}
	if p.Y != "" {
		parts = append(parts, "y="+p.Y)
	}
	if p.Color != "" {
		parts = append(parts, "color="+p.Color)
	}
	if p.Eval != "" {
		parts = append(parts, "eval="+p.Eval)
	}
	if p.Aspect != "" {
		parts = append(parts, "aspect="+p.Aspect)
	}

	if len(parts) == 0 {
		return "pad"
	}
	return "pad=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (p *Pad) Err() error {
	if p.err != nil {
		return p.err
	}
	return p.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -vf.
func (p *Pad) ProvideOption() options.Option {
	if p.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: p.String()}
}