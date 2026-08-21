package aresample

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// Aresample представляет аудиофильтр aresample из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#aresample
type Aresample struct {
	// SampleRate — частота дискретизации, задаётся как отдельное число (например, "48000").
	// Может быть также выражением. Если не указана, фильтр автоматически преобразует между входом и выходом.
	SampleRate string `json:"sample_rate,omitempty"`
	// Options — дополнительные опции ресемплера в формате key=value (например, "async": "1000").
	// Полный список см. в документации ffmpeg-resampler.
	Options map[string]string `json:"options,omitempty"`

	err error
}

// New создаёт фильтр Aresample и применяет переданные опции.
// Поддерживаются опции: "sample_rate", "rate", "r" (синонимы) — устанавливают SampleRate.
// Все остальные опции сохраняются как параметры ресемплера.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Aresample {
	a := &Aresample{
		Options: make(map[string]string),
	}
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
func (a *Aresample) apply(opt options.Option) error {
	switch opt.Key {
	case "sample_rate", "rate", "r":
		if opt.Value == "" {
			return fmt.Errorf("aresample: sample rate cannot be empty")
		}
		a.SampleRate = opt.Value
	default:
		if opt.Key == "" {
			return fmt.Errorf("aresample: option key cannot be empty")
		}
		a.Options[opt.Key] = opt.Value
	}
	return nil
}

// Validate проверяет целостность установленных полей.
// В настоящее время дополнительных проверок нет.
func (a *Aresample) Validate() error {
	return nil
}

// String возвращает строку фильтра (например, "aresample=48000" или "aresample=async=1000").
// При ошибке возвращает пустую строку.
func (a *Aresample) String() string {
	if a.Err() != nil {
		return ""
	}
	var parts []string
	if a.SampleRate != "" {
		parts = append(parts, a.SampleRate)
	}
	// Для детерминированного порядка сортируем ключи
	keys := make([]string, 0, len(a.Options))
	for k := range a.Options {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, k+"="+a.Options[k])
	}
	if len(parts) == 0 {
		return "aresample"
	}
	return "aresample=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (a *Aresample) Err() error {
	if a.err != nil {
		return a.err
	}
	return a.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -af.
func (a *Aresample) ProvideOption() options.Option {
	if a.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-af", Value: a.String()}
}