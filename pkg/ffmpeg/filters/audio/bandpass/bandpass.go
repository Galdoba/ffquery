package bandpass

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
	"github.com/Galdoba/ffcmd/ffmpeg/utils"
)

// Bandpass представляет аудиофильтр bandpass из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#bandpass
type Bandpass struct {
	// Frequency, F — центральная частота фильтра. По умолчанию 3000 Гц.
	Frequency *float64 `json:"frequency,omitempty"`
	// Csg — если true, постоянное усиление skirt (peak gain = Q). По умолчанию false.
	Csg *bool `json:"csg,omitempty"`
	// WidthType, T — метод задания полосы пропускания: "h", "q", "o", "s", "k".
	// h — Гц, q — Q-фактор, o — октава, s — slope, k — кГц.
	WidthType string `json:"width_type,omitempty"`
	// Width, W — ширина полосы в единицах WidthType.
	Width *float64 `json:"width,omitempty"`
	// Mix, M — доля отфильтрованного сигнала в выходе. Диапазон [0,1], по умолчанию 1.
	Mix *float64 `json:"mix,omitempty"`
	// Channels, C — какие каналы фильтровать. По умолчанию все доступные.
	Channels string `json:"channels,omitempty"`
	// Normalize, N — нормализовать коэффициенты биквада. По умолчанию false.
	Normalize *bool `json:"normalize,omitempty"`
	// Transform, A — тип преобразования IIR фильтра: "di", "dii", "tdi", "tdii", "latt", "svf", "zdf".
	Transform string `json:"transform,omitempty"`
	// Precision, R — точность фильтрации: "auto", "s16", "s32", "f32", "f64".
	Precision string `json:"precision,omitempty"`
	// BlockSize, B — размер блока для обратного IIR-процессинга. Должен быть >= 0.
	BlockSize *int `json:"block_size,omitempty"`

	err error
}

// New создаёт фильтр Bandpass и применяет переданные опции.
// Поддерживаются опции: frequency/f, csg, width_type/t, width/w, mix/m,
// channels/c, normalize/n, transform/a, precision/r, block_size/b.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Bandpass {
	b := &Bandpass{}
	for _, opt := range opts {
		if err := b.apply(opt); err != nil {
			b.err = err
			break
		}
	}
	if b.err == nil {
		b.err = b.Validate()
	}
	return b
}

func (b *Bandpass) apply(opt options.Option) error {
	switch opt.Key {
	case "frequency", "f":
		v, err := strconv.ParseFloat(opt.Value, 64)
		if err != nil {
			return fmt.Errorf("bandpass: frequency must be a number, got %q", opt.Value)
		}
		if v <= 0 {
			return fmt.Errorf("bandpass: frequency must be positive, got %v", v)
		}
		b.Frequency = &v
	case "csg":
		v, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("bandpass: csg must be boolean, got %q", opt.Value)
		}
		b.Csg = &v
	case "width_type", "t":
		if !isValidWidthType(opt.Value) {
			return fmt.Errorf("bandpass: width_type must be one of h, q, o, s, k, got %q", opt.Value)
		}
		b.WidthType = opt.Value
	case "width", "w":
		v, err := strconv.ParseFloat(opt.Value, 64)
		if err != nil {
			return fmt.Errorf("bandpass: width must be a number, got %q", opt.Value)
		}
		if v <= 0 {
			return fmt.Errorf("bandpass: width must be positive, got %v", v)
		}
		b.Width = &v
	case "mix", "m":
		v, err := utils.ParseFloat64Bounded(opt.Value, 0, 1)
		if err != nil {
			return fmt.Errorf("bandpass: mix must be in [0,1], got %q: %w", opt.Value, err)
		}
		b.Mix = &v
	case "channels", "c":
		if opt.Value == "" {
			return fmt.Errorf("bandpass: channels cannot be empty")
		}
		b.Channels = opt.Value
	case "normalize", "n":
		v, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("bandpass: normalize must be boolean, got %q", opt.Value)
		}
		b.Normalize = &v
	case "transform", "a":
		if !isValidTransform(opt.Value) {
			return fmt.Errorf("bandpass: transform must be one of di, dii, tdi, tdii, latt, svf, zdf, got %q", opt.Value)
		}
		b.Transform = opt.Value
	case "precision", "r":
		if !isValidPrecision(opt.Value) {
			return fmt.Errorf("bandpass: precision must be one of auto, s16, s32, f32, f64, got %q", opt.Value)
		}
		b.Precision = opt.Value
	case "block_size", "b":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 0 {
			return fmt.Errorf("bandpass: block_size must be a non-negative integer, got %q", opt.Value)
		}
		b.BlockSize = &n
	default:
		return fmt.Errorf("bandpass: unknown option %q", opt.Key)
	}
	return nil
}

func isValidWidthType(v string) bool {
	switch v {
	case "h", "q", "o", "s", "k":
		return true
	}
	return false
}

func isValidTransform(v string) bool {
	switch v {
	case "di", "dii", "tdi", "tdii", "latt", "svf", "zdf":
		return true
	}
	return false
}

func isValidPrecision(v string) bool {
	switch v {
	case "auto", "s16", "s32", "f32", "f64":
		return true
	}
	return false
}

// Validate проверяет целостность установленных полей.
// Дополнительных проверок нет, так как все валидации выполнены в apply.
func (b *Bandpass) Validate() error {
	return nil
}

// String возвращает строку фильтра (например, "bandpass=frequency=1000:width_type=h:width=200").
// При ошибке возвращает пустую строку.
func (b *Bandpass) String() string {
	if b.Err() != nil {
		return ""
	}
	var parts []string

	if b.Frequency != nil {
		parts = append(parts, fmt.Sprintf("frequency=%v", *b.Frequency))
	}
	if b.Csg != nil {
		if *b.Csg {
			parts = append(parts, "csg=1")
		} else {
			parts = append(parts, "csg=0")
		}
	}
	if b.WidthType != "" {
		parts = append(parts, "width_type="+b.WidthType)
	}
	if b.Width != nil {
		parts = append(parts, fmt.Sprintf("width=%v", *b.Width))
	}
	if b.Mix != nil {
		parts = append(parts, fmt.Sprintf("mix=%v", *b.Mix))
	}
	if b.Channels != "" {
		parts = append(parts, "channels="+b.Channels)
	}
	if b.Normalize != nil {
		if *b.Normalize {
			parts = append(parts, "normalize=1")
		} else {
			parts = append(parts, "normalize=0")
		}
	}
	if b.Transform != "" {
		parts = append(parts, "transform="+b.Transform)
	}
	if b.Precision != "" {
		parts = append(parts, "precision="+b.Precision)
	}
	if b.BlockSize != nil {
		parts = append(parts, fmt.Sprintf("block_size=%d", *b.BlockSize))
	}

	if len(parts) == 0 {
		return "bandpass"
	}
	return "bandpass=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (b *Bandpass) Err() error {
	if b.err != nil {
		return b.err
	}
	return b.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -af.
func (b *Bandpass) ProvideOption() options.Option {
	if b.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-af", Value: b.String()}
}