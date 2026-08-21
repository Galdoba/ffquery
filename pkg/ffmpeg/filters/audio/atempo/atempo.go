package atempo

import (
	"fmt"
	"strconv"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// Atempo представляет аудиофильтр atempo из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#atempo
type Atempo struct {
	// Tempo — скорость воспроизведения. Допустимый диапазон [0.5, 100.0].
	// Может быть числом или выражением (например, "sqrt(3)").
	// Если не задано, фильтр предполагает номинальное значение 1.0.
	Tempo string `json:"tempo,omitempty"`

	err error
}

// New создаёт фильтр Atempo и применяет переданные опции.
// Поддерживается только опция "tempo".
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Atempo {
	a := &Atempo{}
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

// apply валидирует опцию и устанавливает соответствующее поле.
func (a *Atempo) apply(opt options.Option) error {
	switch opt.Key {
	case "tempo":
		if opt.Value == "" {
			return fmt.Errorf("atempo: tempo value cannot be empty")
		}
		a.Tempo = opt.Value
	default:
		return fmt.Errorf("atempo: unknown option %q", opt.Key)
	}
	return nil
}

// Validate проверяет целостность установленных полей.
// Если Tempo является числовым литералом, проверяет диапазон [0.5, 100.0].
// Если это выражение, проверка невозможна на этом этапе.
func (a *Atempo) Validate() error {
	if a.Tempo == "" {
		return nil
	}
	if f, err := strconv.ParseFloat(a.Tempo, 64); err == nil {
		if f < 0.5 || f > 100.0 {
			return fmt.Errorf("atempo: tempo must be in [0.5, 100.0], got %v", f)
		}
	}
	return nil
}

// String возвращает строку фильтра (например, "atempo=0.8" или "atempo=sqrt(3)").
// При ошибке возвращает пустую строку.
func (a *Atempo) String() string {
	if a.Err() != nil {
		return ""
	}
	if a.Tempo == "" {
		return "atempo"
	}
	return "atempo=" + a.Tempo
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (a *Atempo) Err() error {
	if a.err != nil {
		return a.err
	}
	return a.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -af.
func (a *Atempo) ProvideOption() options.Option {
	if a.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-af", Value: a.String()}
}
