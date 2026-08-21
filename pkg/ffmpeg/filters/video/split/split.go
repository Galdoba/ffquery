package split

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// Split представляет видеофильтр split из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#split_002c-asplit
type Split struct {
	// Outputs задаёт количество выходных потоков. По умолчанию 2.
	Outputs *int `json:"outputs,omitempty"`

	err error
}

// New создаёт фильтр Split и применяет переданные опции.
// Поддерживается опция "outputs" (или "n").
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Split {
	s := &Split{}
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

func (s *Split) apply(opt options.Option) error {
	switch opt.Key {
	case "outputs", "n":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 1 {
			return fmt.Errorf("split: outputs must be a positive integer, got %q", opt.Value)
		}
		s.Outputs = &n
	default:
		return fmt.Errorf("split: unknown option %q", opt.Key)
	}
	return nil
}

func (s *Split) Validate() error {
	return nil
}

func (s *Split) String() string {
	if s.Err() != nil {
		return ""
	}
	if s.Outputs != nil {
		return "split=" + strconv.Itoa(*s.Outputs)
	}
	return "split"
}

func (s *Split) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.Validate()
}

func (s *Split) ProvideOption() options.Option {
	if s.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: s.String()}
}