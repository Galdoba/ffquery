package filters

import (
	"strings"
)

// Graph представляет набор цепочек, разделённых точкой с запятой.
type Graph struct {
	Chains []*Chain
	err    error
}

// NewGraph создаёт граф. Прекращает добавление цепочек при ошибке.
func NewGraph(chains ...*Chain) *Graph {
	g := &Graph{}
	for _, ch := range chains {
		if ch.Err() != nil {
			g.err = ch.Err()
			break
		}
		g.Chains = append(g.Chains, ch)
	}
	return g
}

// String возвращает строку графа. При ошибке – пустую строку.
func (g *Graph) String() string {
	if g.err != nil {
		return ""
	}
	parts := make([]string, len(g.Chains))
	for i, ch := range g.Chains {
		parts[i] = ch.String()
	}
	return strings.Join(parts, ";")
}

// Err возвращает ошибку графа.
func (g *Graph) Err() error { return g.err }
