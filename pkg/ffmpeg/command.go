package ffmpeg

import "net/url"

type Command struct {
	GlobalOptions     []Option          `json:"global_options,omitempty"`
	InputFileOptions  []Input           `json:"input_file_options,omitempty"`
	OutputFileOptions []Output          `json:"output_file_options,omitempty"`
	globalFieldValues map[string]string `json:"-"` //example outputDirectory=/path/to/directory/
}

func NewCommand(opts ...OptFunc) *Command {
	c := Command{}
	c.globalFieldValues = make(map[string]string)
	return &c
}

type OptFunc func(*Command)

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
	//filter_complex как частный случай опции выходного файла
	// TODO: в перспективе вынести в отдельный пакет
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
