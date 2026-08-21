package asplit

import (
	"fmt"
	"strconv"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// Asplit представляет аудиофильтр asplit из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#split_002c-asplit
type Asplit struct {
	// Outputs задаёт количество выходных потоков. По умолчанию 2.
	Outputs *int `json:"outputs,omitempty"`

	err error
}

// New создаёт фильтр Asplit и применяет переданные опции.
// Поддерживается опция "outputs" (или "n").
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Asplit {
	a := &Asplit{}
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

func (a *Asplit) apply(opt options.Option) error {
	switch opt.Key {
	case "outputs", "n":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 1 {
			return fmt.Errorf("asplit: outputs must be a positive integer, got %q", opt.Value)
		}
		a.Outputs = &n
	default:
		return fmt.Errorf("asplit: unknown option %q", opt.Key)
	}
	return nil
}

func (a *Asplit) Validate() error {
	return nil
}

func (a *Asplit) String() string {
	if a.Err() != nil {
		return ""
	}
	if a.Outputs != nil {
		return "asplit=" + strconv.Itoa(*a.Outputs)
	}
	return "asplit"
}

func (a *Asplit) Err() error {
	if a.err != nil {
		return a.err
	}
	return a.Validate()
}

func (a *Asplit) ProvideOption() options.Option {
	if a.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-af", Value: a.String()}
}