package options

type Global struct {
	Key  string `json:"key,omitempty"`  // option name without leading dash
	Args []*Arg `json:"args,omitempty"` // arguments (tokens) for this option
}

type Arg struct {
	Value     string `json:"value,omitempty"`     // simple leaf value
	Key       string `json:"key,omitempty"`       // for key=value pairs (left side)
	ValueArg  *Arg   `json:"value_arg,omitempty"` // right side of key=value (allows nesting)
	Items     []*Arg `json:"items,omitempty"`     // list of sub-args joined by Delimiter
	Delimiter string `json:"delimiter,omitempty"` // e.g. ":", ",", "|" (default space if items used)
	Prefix    string `json:"prefix,omitempty"`    // optional prefix added before serializing (rare)
}
