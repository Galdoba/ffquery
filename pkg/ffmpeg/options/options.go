package options

// Option представляет опцию командной строки ffmpeg.
type Option struct {
	Key         string
	Value       string
	Description string
}

// ProvideOption реализует интерфейс OptionProvider.
func (o Option) ProvideOption() Option { return o }

// Err возвращает nil (у простой опции ошибок нет).
func (o Option) Err() error { return nil }

// String возвращает строковое представление опции (для отладки).
func (o Option) String() string {
	if o.Value == "" {
		return o.Key
	}
	return o.Key + " " + o.Value
}

// Opt создаёт простую опцию с ключом и значением.
func Opt(key, value string) Option {
	return Option{Key: key, Value: value}
}

// Flag создаёт опцию-флаг (без значения).
func Flag(key string) Option {
	return Option{Key: key}
}

// OptionProvider – интерфейс для объектов, способных предоставить опцию.
type OptionProvider interface {
	ProvideOption() Option
	Err() error
}
