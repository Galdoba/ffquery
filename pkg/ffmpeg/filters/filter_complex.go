package filters

import "github.com/Galdoba/ffcmd/ffmpeg/options"

// FilterComplex оборачивает Graph как опцию -filter_complex.
type FilterComplex struct {
	Graph *Graph
}

// ProvideOption возвращает опцию -filter_complex.
func (fc FilterComplex) ProvideOption() options.Option {
	return options.Option{Key: "-filter_complex", Value: fc.Graph.String()}
}

// Err возвращает ошибку графа.
func (fc FilterComplex) Err() error {
	return fc.Graph.Err()
}

// String возвращает строковое представление опции -filter_complex.
func (fc FilterComplex) String() string {
	if fc.Err() != nil {
		return ""
	}
	return "-filter_complex " + fc.Graph.String()
}
