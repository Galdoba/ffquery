package unsharp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// Unsharp представляет видеофильтр unsharp из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#unsharp
type Unsharp struct {
	// LumaMsizeX, Lx — горизонтальный размер матрицы яркости. Нечетное целое от 3 до 23. По умолчанию 5.
	LumaMsizeX *int `json:"luma_msize_x,omitempty"`
	// LumaMsizeY, Ly — вертикальный размер матрицы яркости. Нечетное целое от 3 до 23. По умолчанию 5.
	LumaMsizeY *int `json:"luma_msize_y,omitempty"`
	// LumaAmount, La — сила эффекта яркости. Число с плавающей точкой, разумные значения от -1.5 до 1.5.
	// Отрицательные размывают, положительные увеличивают резкость, 0 отключает. По умолчанию 1.0.
	LumaAmount *float64 `json:"luma_amount,omitempty"`
	// ChromaMsizeX, Cx — горизонтальный размер матрицы цветности. Нечетное целое от 3 до 23. По умолчанию 5.
	ChromaMsizeX *int `json:"chroma_msize_x,omitempty"`
	// ChromaMsizeY, Cy — вертикальный размер матрицы цветности. Нечетное целое от 3 до 23. По умолчанию 5.
	ChromaMsizeY *int `json:"chroma_msize_y,omitempty"`
	// ChromaAmount, Ca — сила эффекта цветности. Число с плавающей точкой, разумные значения от -1.5 до 1.5.
	// Отрицательные размывают, положительные увеличивают резкость, 0 отключает. По умолчанию 0.0.
	ChromaAmount *float64 `json:"chroma_amount,omitempty"`
	// AlphaMsizeX, Ax — горизонтальный размер матрицы альфа-канала. Нечетное целое от 3 до 23. По умолчанию 5.
	AlphaMsizeX *int `json:"alpha_msize_x,omitempty"`
	// AlphaMsizeY, Ay — вертикальный размер матрицы альфа-канала. Нечетное целое от 3 до 23. По умолчанию 5.
	AlphaMsizeY *int `json:"alpha_msize_y,omitempty"`
	// AlphaAmount, Aa — сила эффекта альфа-канала. Число с плавающей точкой, разумные значения от -1.5 до 1.5.
	// Отрицательные размывают, положительные увеличивают резкость, 0 отключает. По умолчанию 0.0.
	AlphaAmount *float64 `json:"alpha_amount,omitempty"`

	err error
}

// New создаёт фильтр Unsharp и применяет переданные опции.
// Поддерживаются опции с синонимами:
//
//	luma_msize_x / lx, luma_msize_y / ly, luma_amount / la,
//	chroma_msize_x / cx, chroma_msize_y / cy, chroma_amount / ca,
//	alpha_msize_x / ax, alpha_msize_y / ay, alpha_amount / aa.
//
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Unsharp {
	u := &Unsharp{}
	for _, opt := range opts {
		if err := u.apply(opt); err != nil {
			u.err = err
			break
		}
	}
	if u.err == nil {
		u.err = u.Validate()
	}
	return u
}

// apply валидирует опцию и устанавливает соответствующее поле.
func (u *Unsharp) apply(opt options.Option) error {
	switch opt.Key {
	case "luma_msize_x", "lx":
		v, err := parseIntSize(opt.Value)
		if err != nil {
			return fmt.Errorf("unsharp: invalid luma_msize_x: %v", err)
		}
		u.LumaMsizeX = &v
	case "luma_msize_y", "ly":
		v, err := parseIntSize(opt.Value)
		if err != nil {
			return fmt.Errorf("unsharp: invalid luma_msize_y: %v", err)
		}
		u.LumaMsizeY = &v
	case "luma_amount", "la":
		v, err := strconv.ParseFloat(opt.Value, 64)
		if err != nil {
			return fmt.Errorf("unsharp: invalid luma_amount %q", opt.Value)
		}
		u.LumaAmount = &v
	case "chroma_msize_x", "cx":
		v, err := parseIntSize(opt.Value)
		if err != nil {
			return fmt.Errorf("unsharp: invalid chroma_msize_x: %v", err)
		}
		u.ChromaMsizeX = &v
	case "chroma_msize_y", "cy":
		v, err := parseIntSize(opt.Value)
		if err != nil {
			return fmt.Errorf("unsharp: invalid chroma_msize_y: %v", err)
		}
		u.ChromaMsizeY = &v
	case "chroma_amount", "ca":
		v, err := strconv.ParseFloat(opt.Value, 64)
		if err != nil {
			return fmt.Errorf("unsharp: invalid chroma_amount %q", opt.Value)
		}
		u.ChromaAmount = &v
	case "alpha_msize_x", "ax":
		v, err := parseIntSize(opt.Value)
		if err != nil {
			return fmt.Errorf("unsharp: invalid alpha_msize_x: %v", err)
		}
		u.AlphaMsizeX = &v
	case "alpha_msize_y", "ay":
		v, err := parseIntSize(opt.Value)
		if err != nil {
			return fmt.Errorf("unsharp: invalid alpha_msize_y: %v", err)
		}
		u.AlphaMsizeY = &v
	case "alpha_amount", "aa":
		v, err := strconv.ParseFloat(opt.Value, 64)
		if err != nil {
			return fmt.Errorf("unsharp: invalid alpha_amount %q", opt.Value)
		}
		u.AlphaAmount = &v
	default:
		return fmt.Errorf("unsharp: unknown option %q", opt.Key)
	}
	return nil
}

// parseIntSize проверяет, что значение является нечетным целым от 3 до 23.
func parseIntSize(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if n < 3 || n > 23 {
		return 0, fmt.Errorf("must be an odd integer between 3 and 23, got %d", n)
	}
	if n%2 == 0 {
		return 0, fmt.Errorf("must be odd, got %d", n)
	}
	return n, nil
}

// Validate проверяет целостность установленных полей.
// Дополнительных проверок не требуется, так как все валидации выполнены в apply.
func (u *Unsharp) Validate() error {
	return nil
}

// String возвращает строку фильтра (например, "unsharp=7:7:2.5").
// При ошибке возвращает пустую строку.
func (u *Unsharp) String() string {
	if u.Err() != nil {
		return ""
	}
	var parts []string

	if u.LumaMsizeX != nil {
		parts = append(parts, fmt.Sprintf("luma_msize_x=%d", *u.LumaMsizeX))
	}
	if u.LumaMsizeY != nil {
		parts = append(parts, fmt.Sprintf("luma_msize_y=%d", *u.LumaMsizeY))
	}
	if u.LumaAmount != nil {
		parts = append(parts, fmt.Sprintf("luma_amount=%v", *u.LumaAmount))
	}
	if u.ChromaMsizeX != nil {
		parts = append(parts, fmt.Sprintf("chroma_msize_x=%d", *u.ChromaMsizeX))
	}
	if u.ChromaMsizeY != nil {
		parts = append(parts, fmt.Sprintf("chroma_msize_y=%d", *u.ChromaMsizeY))
	}
	if u.ChromaAmount != nil {
		parts = append(parts, fmt.Sprintf("chroma_amount=%v", *u.ChromaAmount))
	}
	if u.AlphaMsizeX != nil {
		parts = append(parts, fmt.Sprintf("alpha_msize_x=%d", *u.AlphaMsizeX))
	}
	if u.AlphaMsizeY != nil {
		parts = append(parts, fmt.Sprintf("alpha_msize_y=%d", *u.AlphaMsizeY))
	}
	if u.AlphaAmount != nil {
		parts = append(parts, fmt.Sprintf("alpha_amount=%v", *u.AlphaAmount))
	}

	if len(parts) == 0 {
		return "unsharp"
	}
	return "unsharp=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (u *Unsharp) Err() error {
	if u.err != nil {
		return u.err
	}
	return u.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -vf.
func (u *Unsharp) ProvideOption() options.Option {
	if u.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: u.String()}
}
