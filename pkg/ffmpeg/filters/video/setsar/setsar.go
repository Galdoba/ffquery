package setsar

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// SetSar представляет видеофильтр setsar из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#setdar_002c-setsar
type SetSar struct {
	// Sar задаёт Sample Aspect Ratio (SAR). Может быть числом с плавающей точкой, выражением или рациональным числом (например, "10/11").
	// Если не задано, используется значение "0" (то же, что и входное).
	Sar string `json:"sar,omitempty"`
	// Max задаёт максимальное целое значение для числителя и знаменателя при приведении соотношения к рациональному.
	// По умолчанию 100. Допустимы только положительные целые.
	Max *int `json:"max,omitempty"`

	err error
}

// New создаёт фильтр SetSar и применяет переданные опции.
// Поддерживаются опции: "r", "ratio", "sar" (синонимы) и "max".
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *SetSar {
	s := &SetSar{}
	for _, opt := range opts {
		if err := s.apply(opt); err != nil {
			s.err = err
			break
		}
	}
	if s.err == nil {
		s.err = s.Validate()
	}
	return s
}

// apply валидирует опцию и устанавливает соответствующее поле.
func (s *SetSar) apply(opt options.Option) error {
	switch opt.Key {
	case "r", "ratio", "sar":
		if opt.Value == "" {
			return fmt.Errorf("setsar: sar/ratio value cannot be empty")
		}
		s.Sar = opt.Value
	case "max":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n <= 0 {
			return fmt.Errorf("setsar: max must be a positive integer, got %q", opt.Value)
		}
		s.Max = &n
	default:
		return fmt.Errorf("setsar: unknown option %q", opt.Key)
	}
	return nil
}

// Validate проверяет целостность установленных полей.
// В настоящее время дополнительных проверок нет, так как все значения валидны.
func (s *SetSar) Validate() error {
	// Например, можно проверить, что Max не установлен без Sar? Но это допустимо.
	return nil
}

// String возвращает строку фильтра (например, "setsar=sar=10/11").
// При ошибке возвращает пустую строку.
func (s *SetSar) String() string {
	if s.Err() != nil {
		return ""
	}
	var parts []string

	if s.Sar != "" {
		parts = append(parts, "sar="+s.Sar)
	}
	if s.Max != nil {
		parts = append(parts, fmt.Sprintf("max=%d", *s.Max))
	}

	if len(parts) == 0 {
		return "setsar"
	}
	return "setsar=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (s *SetSar) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -vf.
func (s *SetSar) ProvideOption() options.Option {
	if s.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: s.String()}
}