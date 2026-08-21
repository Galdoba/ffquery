package metadata

import (
	"fmt"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

// Metadata представляет видеофильтр metadata из FFmpeg.
// Документация: https://ffmpeg.org/ffmpeg-all.html#metadata_002c-ametadata
type Metadata struct {
	// Mode — режим работы фильтра: "select", "add", "modify", "delete", "print".
	Mode string `json:"mode,omitempty"`
	// Key — ключ метаданных. Обязателен для всех режимов, кроме print и delete.
	Key string `json:"key,omitempty"`
	// Value — значение метаданных. Обязателен для режимов add и modify.
	Value string `json:"value,omitempty"`
	// Function — функция сравнения: "same_str", "starts_with", "less", "equal", "greater", "expr", "ends_with".
	Function string `json:"function,omitempty"`
	// Expr — выражение для function=expr. Может содержать VALUE1, VALUE2.
	Expr string `json:"expr,omitempty"`
	// File — файл для вывода в режиме print. Если не задан, вывод в лог.
	File string `json:"file,omitempty"`
	// Direct — уменьшить буферизацию при выводе в URL (print mode).
	Direct *bool `json:"direct,omitempty"`

	err error
}

// New создаёт фильтр Metadata и применяет переданные опции.
// Поддерживаются опции: mode, key, value, function, expr, file, direct.
// При ошибке в любой опции сохраняет её и прекращает обработку.
func New(opts ...options.Option) *Metadata {
	m := &Metadata{}
	for _, opt := range opts {
		if err := m.apply(opt); err != nil {
			m.err = err
			break
		}
	}
	if m.err == nil {
		m.err = m.Validate()
	}
	return m
}

func (m *Metadata) apply(opt options.Option) error {
	switch opt.Key {
	case "mode":
		if !isValidMode(opt.Value) {
			return fmt.Errorf("metadata: mode must be one of select, add, modify, delete, print, got %q", opt.Value)
		}
		m.Mode = opt.Value
	case "key":
		if opt.Value == "" {
			return fmt.Errorf("metadata: key cannot be empty")
		}
		m.Key = opt.Value
	case "value":
		m.Value = opt.Value
	case "function":
		if !isValidFunction(opt.Value) {
			return fmt.Errorf("metadata: function must be one of same_str, starts_with, less, equal, greater, expr, ends_with, got %q", opt.Value)
		}
		m.Function = opt.Value
	case "expr":
		if opt.Value == "" {
			return fmt.Errorf("metadata: expr cannot be empty")
		}
		m.Expr = opt.Value
	case "file":
		m.File = opt.Value
	case "direct":
		b, err := parseBool(opt.Value)
		if err != nil {
			return fmt.Errorf("metadata: direct must be boolean, got %q", opt.Value)
		}
		m.Direct = &b
	default:
		return fmt.Errorf("metadata: unknown option %q", opt.Key)
	}
	return nil
}

func isValidMode(v string) bool {
	switch v {
	case "select", "0", "add", "1", "modify", "2", "delete", "3", "print", "4":
		return true
	}
	return false
}

func isValidFunction(v string) bool {
	switch v {
	case "same_str", "starts_with", "less", "equal", "greater", "expr", "ends_with":
		return true
	}
	return false
}

// Validate проверяет согласованность опций.
func (m *Metadata) Validate() error {
	if m.Mode == "" {
		return fmt.Errorf("metadata: mode is required")
	}
	// key обязателен для всех режимов, кроме print и delete
	if m.Mode != "print" && m.Mode != "delete" && m.Key == "" {
		return fmt.Errorf("metadata: key is required for mode %q", m.Mode)
	}
	// value обязателен для add и modify
	if (m.Mode == "add" || m.Mode == "modify") && m.Value == "" {
		return fmt.Errorf("metadata: value is required for mode %q", m.Mode)
	}
	// expr обязателен для function=expr
	if m.Function == "expr" && m.Expr == "" {
		return fmt.Errorf("metadata: expr is required when function=expr")
	}
	return nil
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", value)
	}
}

// String возвращает строку фильтра (например, "metadata=mode=print:key=lavfi.signalstats.YDIF").
// При ошибке возвращает пустую строку.
func (m *Metadata) String() string {
	if m.Err() != nil {
		return ""
	}
	var parts []string
	parts = append(parts, "mode="+m.Mode)
	if m.Key != "" {
		parts = append(parts, "key="+m.Key)
	}
	if m.Value != "" {
		parts = append(parts, "value="+m.Value)
	}
	if m.Function != "" {
		parts = append(parts, "function="+m.Function)
	}
	if m.Expr != "" {
		parts = append(parts, "expr="+m.Expr)
	}
	if m.File != "" {
		parts = append(parts, "file="+m.File)
	}
	if m.Direct != nil {
		if *m.Direct {
			parts = append(parts, "direct=1")
		} else {
			parts = append(parts, "direct=0")
		}
	}
	return "metadata=" + strings.Join(parts, ":")
}

// Err возвращает ошибку, возникшую при конструировании или валидации.
func (m *Metadata) Err() error {
	if m.err != nil {
		return m.err
	}
	return m.Validate()
}

// ProvideOption реализует options.OptionProvider. Возвращает опцию -vf.
func (m *Metadata) ProvideOption() options.Option {
	if m.Err() != nil {
		return options.Option{}
	}
	return options.Option{Key: "-vf", Value: m.String()}
}
