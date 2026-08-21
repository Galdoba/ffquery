package amerge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// Amerge представляет аудиофильтр amerge из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#amerge
type Amerge struct {
	// Inputs задаёт количество входных аудиопотоков. По умолчанию 2.
	Inputs *int `json:"inputs,omitempty"`
	// LayoutMode управляет определением выходной раскладки каналов.
	// Возможные значения: "legacy" (по умолчанию), "reset", "normal".
	LayoutMode string `json:"layout_mode,omitempty"`

	err error
}

// New создаёт фильтр Amerge и применяет переданные опции.
// Поддерживаются опции: "inputs", "layout_mode".
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Amerge {
	a := &Amerge{}
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

func (a *Amerge) apply(opt options.Option) error {
	switch opt.Key {
	case "inputs":
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 1 {
			return fmt.Errorf("amerge: inputs must be a positive integer, got %q", opt.Value)
		}
		a.Inputs = &n
	case "layout_mode":
		if opt.Value != "legacy" && opt.Value != "reset" && opt.Value != "normal" {
			return fmt.Errorf("amerge: layout_mode must be 'legacy', 'reset' or 'normal', got %q", opt.Value)
		}
		a.LayoutMode = opt.Value
	default:
		return fmt.Errorf("amerge: unknown option %q", opt.Key)
	}
	return nil
}

func (a *Amerge) Validate() error {
	return nil
}

func (a *Amerge) String() string {
	if a.Err() != nil {
		return ""
	}
	var parts []string
	if a.Inputs != nil {
		parts = append(parts, "inputs="+strconv.Itoa(*a.Inputs))
	}
	if a.LayoutMode != "" {
		parts = append(parts, "layout_mode="+a.LayoutMode)
	}
	if len(parts) == 0 {
		return "amerge"
	}
	return "amerge=" + strings.Join(parts, ":")
}

func (a *Amerge) Err() error {
	if a.err != nil {
		return a.err
	}
	return a.Validate()
}

func (a *Amerge) ProvideOption() options.Option {
	if a.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-af", Value: a.String()}
}