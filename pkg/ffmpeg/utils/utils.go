package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// NormalizeOptionKey приводит ключ опции к каноническому виду: нижний регистр, ведущий "-".
func NormalizeOptionKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if !strings.HasPrefix(key, "-") {
		key = "-" + key
	}
	return key
}

// NormalizeFilterOptionKey приводит ключ опции фильтра к нижнему регистру без дефисов.
func NormalizeFilterOptionKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.TrimPrefix(key, "-")
	return key
}

// ParseBool преобразует строковое представление в bool.
func ParseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", value)
	}
}

// ParseFloat64Bounded парсит строку в float64 и проверяет, что значение находится в диапазоне [low, high].
// Если low > high, границы автоматически меняются местами.
func ParseFloat64Bounded(value string, low, high float64) (float64, error) {
	if low > high {
		low, high = high, low
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float value %q", value)
	}
	if f < low || f > high {
		return 0, fmt.Errorf("value %v out of range [%v, %v]", f, low, high)
	}
	return f, nil
}

// EscapeFilterValue экранирует специальные символы в значении фильтра FFmpeg.
func EscapeFilterValue(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`:`, `\:`,
		`,`, `\,`,
		`'`, `\'`,
		`[`, `\[`,
		`]`, `\]`,
	)
	return replacer.Replace(s)
}

// ValidateEnum проверяет, входит ли значение в список допустимых.
func ValidateEnum(value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("invalid value %q: expected one of %v", value, allowed)
}
