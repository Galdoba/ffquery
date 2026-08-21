package ffmpeg

import (
	"fmt"
	"net/url"
)

type Command struct {
	GlobalOptions     []Option `json:"global_options,omitempty"`
	Input             []Input  `json:"input_file_options,omitempty"`
	FilterComplex     Option
	Lavfi             Option
	Output            []Output          `json:"output_file_options,omitempty"`
	globalFieldValues map[string]string `json:"-"` //example outputDirectory=/path/to/directory/
	buildError        error
}

func NewCommand() *Command {
	c := Command{}
	c.globalFieldValues = make(map[string]string)
	return &c
}

func (cmd *Command) AddInput(inputs ...Input) *Command {
	if cmd.buildError != nil {
		return cmd
	}
	if len(cmd.Input) > 0 {
		cmd.buildError = fmt.Errorf("input added multiple times")
		return cmd
	}
	for _, in := range inputs {
		cmd.Input = append(cmd.Input, in)
	}
	return cmd
}

func (cmd *Command) AddOutput(outputs ...Output) *Command {
	if cmd.buildError != nil {
		return cmd
	}
	if len(cmd.Output) > 0 {
		cmd.buildError = fmt.Errorf("output added multiple times")
		return cmd
	}
	for _, out := range outputs {
		cmd.Output = append(cmd.Output, out)
	}
	return cmd
}

func (cmd *Command) AddOptions(options ...Option) *Command {
	if cmd.buildError != nil {
		return cmd
	}
	if len(cmd.GlobalOptions) > 0 {
		cmd.buildError = fmt.Errorf("options added multiple times")
		return cmd
	}
	for _, opt := range options {
		if opt.Key == "-filter_complex" {
			continue
		}
		cmd.GlobalOptions = append(cmd.GlobalOptions, opt)
	}
	return cmd
}

func (cmd *Command) AddFiltercomplex(fc Option) *Command {
	if cmd.buildError != nil {
		return cmd
	}
	if cmd.FilterComplex.Value != "" {
		cmd.buildError = fmt.Errorf("filter_complex added multiple times")
		return cmd
	}
	cmd.FilterComplex = fc
	return cmd
}

func (cmd *Command) Build() ([]string, error) {
	if cmd.buildError != nil {
		return nil, cmd.buildError
	}
	//do magic
	// return command args or exec.Cmd
	return []string{"ffmpeg", "with", "args"}, nil
}

type Option struct {
	Key         string
	Value       string
	Description string
}

type Input struct {
	Options  []Option
	InputUrl Filepath
}

type Output struct {
	Options   []Option
	OutputUrl Filepath
}

// будем использовать для простоты, позже переключиться на URL
type Filepath struct {
	url  url.URL
	Dir  string
	Name string
	Ext  string
}
