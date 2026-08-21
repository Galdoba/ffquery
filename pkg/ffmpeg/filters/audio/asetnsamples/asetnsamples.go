package asetnsamples

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
	"github.com/Galdoba/ffcmd/ffmpeg/utils"
)

// Asetnsamples представляет аудиофильтр asetnsamples из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#asetnsamples
type Asetnsamples struct {
	// NbOutSamples, N — количество сэмплов на каждый выходной аудиокадр (на канал).
	// По умолчанию 1024.
	NbOutSamples *int `json:"nb_out_samples,omitempty"`
	// Pad, P — если true, последний кадр дополняется нулями до того же размера.
	// По умолчанию true.
	Pad *bool `json:"pad,omitempty"`

	err error
}

// New создаёт фильтр Asetnsamples и применяет переданные опции.
// Поддерживаются опции: nb_out_samples/n, pad/p.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Asetnsamples {
	a := &Asetnsamples{}
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

func (a *Asetnsamples) apply(opt options.Option) error {
	switch opt.Key {
	case "nb_out_samples", "n":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n <= 0 {
			return fmt.Errorf("asetnsamples: nb_out_samples must be a positive integer, got %q", opt.Value)
		}
		a.NbOutSamples = &n
	case "pad", "p":
		b, err := utils.ParseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("asetnsamples: pad must be boolean, got %q", opt.Value)
		}
		a.Pad = &b
	default:
		return fmt.Errorf("asetnsamples: unknown option %q", opt.Key)
	}
	return nil
}

// Validate проверяет целостность установленных полей.
// Дополнительных проверок нет.
func (a *Asetnsamples) Validate() error {
	return nil
}

// String возвращает строку фильтра (например, "asetnsamples=nb_out_samples=1234:pad=0").
// При ошибке возвращает пустую строку.
func (a *Asetnsamples) String() string {
	if a.Err() != nil {
		return ""
	}
	var parts []string

	if a.NbOutSamples != nil {
		parts = append(parts, fmt.Sprintf("nb_out_samples=%d", *a.NbOutSamples))
	}
	if a.Pad != nil {
		if *a.Pad {
			parts = append(parts, "pad=1")
		} else {
			parts = append(parts, "pad=0")
		}
	}

	if len(parts) == 0 {
		return "asetnsamples"
	}
	return "asetnsamples=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (a *Asetnsamples) Err() error {
	if a.err != nil {
		return a.err
	}
	return a.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -af.
func (a *Asetnsamples) ProvideOption() options.Option {
	if a.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-af", Value: a.String()}
}
