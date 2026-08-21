package input

import "github.com/Galdoba/ffcmd/ffmpeg/options"

// Input представляет входной файл с опциями.
type Input struct {
	Options  []options.Option
	InputURL string
	err      error
}

// NewInput создаёт Input. Прекращает обработку опций при ошибке.
func NewInput(path string, providers ...options.OptionProvider) Input {
	in := Input{InputURL: path}
	for _, p := range providers {
		if in.err != nil {
			break
		}
		if err := p.Err(); err != nil {
			in.err = err
			break
		}
		in.Options = append(in.Options, p.ProvideOption())
	}
	return in
}

// Err возвращает ошибку (если была).
func (i Input) Err() error { return i.err }
