package source

import (
	"fmt"

	"github.com/Galdoba/ffquery/internal/domains/content"
)

type Prefix struct {
	Key  string
	Type content.Type
}

func (p Prefix) String() string {
	return fmt.Sprintf("%s--%v", p.Key, p.Type)
}
