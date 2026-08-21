package output

import "github.com/Galdoba/ffcmd/ffmpeg/options"

// Output представляет выходной файл с опциями.
type Output struct {
	Options   []options.Option
	OutputURL string
	err       error
}

// NewOutput создаёт Output. Прекращает обработку опций при ошибке.
func NewOutput(path string, providers ...options.OptionProvider) Output {
	out := Output{OutputURL: path}
	for _, p := range providers {
		if out.err != nil {
			break
		}
		if err := p.Err(); err != nil {
			out.err = err
			break
		}
		out.Options = append(out.Options, p.ProvideOption())
	}
	return out
}

// Err возвращает ошибку (если была).
func (o Output) Err() error { return o.err }
