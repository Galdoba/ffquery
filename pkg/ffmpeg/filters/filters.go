package filters

import (
	"strings"
)

// Filter – интерфейс для любого фильтра. Любой фильтр должен возвращать строку и ошибку.
type Filter interface {
	String() string
	Err() error
}

// Chain представляет линейную цепочку фильтров.
type Chain struct {
	In      string
	Filters []Filter
	Out     string
	err     error
}

// NewChain создаёт цепочку. Прекращает добавление фильтров, если встретит ошибку.
func NewChain(in string, filters ...Filter) *Chain {
	c := &Chain{In: in}
	for _, f := range filters {
		if f.Err() != nil {
			c.err = f.Err()
			break
		}
		c.Filters = append(c.Filters, f)
	}
	return c
}

// OutTo задаёт выходную метку и возвращает цепочку для дальнейшего использования.
func (c *Chain) OutTo(out string) *Chain {
	c.Out = out
	return c
}

// String собирает строку цепочки. При ошибке возвращает пустую строку.
func (c *Chain) String() string {
	if c.err != nil {
		return ""
	}
	var sb strings.Builder
	if c.In != "" {
		sb.WriteString(c.In)
	}
	for i, f := range c.Filters {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(f.String())
	}
	if c.Out != "" {
		sb.WriteString(c.Out)
	}
	return sb.String()
}

// Err возвращает ошибку цепочки (если есть).
func (c *Chain) Err() error { return c.err }
