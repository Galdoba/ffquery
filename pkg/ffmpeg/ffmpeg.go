package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Galdoba/ffcmd/ffmpeg/filters"
	"github.com/Galdoba/ffcmd/ffmpeg/input"
	"github.com/Galdoba/ffcmd/ffmpeg/options"
	"github.com/Galdoba/ffcmd/ffmpeg/output"
)

type Command struct {
	globalOptions []options.Option
	inputs        []input.Input
	outputs       []output.Output
	filterComplex *options.Option
	lavfi         *options.Option
	buildError    error

	// Дополнительные файлы для exec.Cmd.ExtraFiles
	extraFiles []*os.File

	// Кастомные потоки (nil = использовать os.Stdout/Stderr по умолчанию)
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// NewCommand создаёт пустую команду.
func NewCommand() *Command {
	return &Command{}
}

// AddExtraFile добавляет файл в ExtraFiles будущего exec.Cmd.
// Файлы нумеруются с 0, что соответствует pipe:0, pipe:1 и т.д. в ffmpeg.
func (c *Command) AddExtraFile(f *os.File) *Command {
	c.extraFiles = append(c.extraFiles, f)
	return c
}

// SetStdin устанавливает поток ввода.
func (c *Command) SetStdin(r io.Reader) *Command {
	c.stdin = r
	return c
}

// SetStdout устанавливает поток вывода.
func (c *Command) SetStdout(w io.Writer) *Command {
	c.stdout = w
	return c
}

// SetStderr устанавливает поток ошибок.
func (c *Command) SetStderr(w io.Writer) *Command {
	c.stderr = w
	return c
}

// ToExecCmd создаёт *exec.Cmd без запуска.
func (c *Command) ToExecCmd(ctx context.Context) (*exec.Cmd, error) {
	args, err := c.Build()
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// Подключаем дополнительные файлы
	if len(c.extraFiles) > 0 {
		cmd.ExtraFiles = c.extraFiles
	}

	// Настраиваем потоки (по умолчанию оставляем nil, чтобы exec.Cmd использовал os.Std*)
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr

	return cmd, nil
}

// Run выполняет команду с текущими настройками.
func (c *Command) Run(ctx context.Context) error {
	cmd, err := c.ToExecCmd(ctx)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// RunWithOutput выполняет команду и возвращает объединённый вывод.
// Если задан кастомный Stdout/Stderr, CombinedOutput не будет работать; используйте Run.
func (c *Command) RunWithOutput(ctx context.Context) ([]byte, error) {
	if c.stdout != nil || c.stderr != nil {
		return nil, fmt.Errorf("RunWithOutput cannot be used with custom stdout/stderr")
	}
	cmd, err := c.ToExecCmd(ctx)
	if err != nil {
		return nil, err
	}
	return cmd.CombinedOutput()
}

func (c *Command) AddGlobalOptions(providers ...options.OptionProvider) *Command {
	if c.buildError != nil {
		return c
	}
	for _, p := range providers {
		if err := p.Err(); err != nil {
			c.buildError = err
			return c
		}
		opt := p.ProvideOption()
		if opt.Key == "" {
			continue
		}
		key := normalizeOptionKey(opt.Key)
		if key == "-filter_complex" || key == "-lavfi" {
			c.buildError = fmt.Errorf("use AddFilterComplex or AddLavfi for %s", key)
			return c
		}
		// Сохраняем опцию с нормализованным ключом для последующего использования в Build
		opt.Key = key
		c.globalOptions = append(c.globalOptions, opt)
	}
	return c
}

func (c *Command) AddInput(in input.Input) *Command {
	if c.buildError != nil {
		return c
	}
	if err := in.Err(); err != nil {
		c.buildError = err
		return c
	}
	c.inputs = append(c.inputs, in)
	return c
}

func (c *Command) AddOutput(out output.Output) *Command {
	if c.buildError != nil {
		return c
	}
	if err := out.Err(); err != nil {
		c.buildError = err
		return c
	}
	c.outputs = append(c.outputs, out)
	return c
}

func (c *Command) AddFilterComplex(fc filters.FilterComplex) *Command {
	if c.buildError != nil {
		return c
	}
	if err := fc.Err(); err != nil {
		c.buildError = err
		return c
	}
	opt := fc.ProvideOption()
	c.filterComplex = &opt
	return c
}

func (c *Command) AddLavfi(fc filters.FilterComplex) *Command {
	if c.buildError != nil {
		return c
	}
	if err := fc.Err(); err != nil {
		c.buildError = err
		return c
	}
	opt := fc.ProvideOption()
	opt.Key = "-lavfi"
	c.lavfi = &opt
	return c
}

func (c *Command) Build() ([]string, error) {
	if c.buildError != nil {
		return nil, c.buildError
	}
	var args []string

	// Глобальные опции
	for _, opt := range c.globalOptions {
		addOptionToArgs(&args, opt)
	}

	// Входы
	for _, in := range c.inputs {
		for _, opt := range in.Options {
			addOptionToArgs(&args, opt)
		}
		args = append(args, "-i", in.InputURL)
	}

	// filter_complex
	if c.filterComplex != nil {
		addOptionToArgs(&args, *c.filterComplex)
	}

	// lavfi
	if c.lavfi != nil {
		addOptionToArgs(&args, *c.lavfi)
	}

	// Выходы
	for _, out := range c.outputs {
		for _, opt := range out.Options {
			addOptionToArgs(&args, opt)
		}
		args = append(args, out.OutputURL)
	}

	return args, nil
}

func (c *Command) String() string {
	args, err := c.Build()
	if err != nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("ffmpeg")
	for i, arg := range args {
		sb.WriteString(" ")
		if i > 0 && (args[i-1] == "-filter_complex" || args[i-1] == "-lavfi") {
			// Оборачиваем значение filter_complex/lavfi в двойные кавычки для shell
			sb.WriteString(`"` + arg + `"`)
		} else {
			sb.WriteString(arg)
		}
	}
	return sb.String()
}

// addOptionToArgs добавляет опцию в слайс аргументов, нормализуя ключ.
func addOptionToArgs(args *[]string, opt options.Option) {
	key := normalizeOptionKey(opt.Key)
	*args = append(*args, key)
	if opt.Value != "" {
		*args = append(*args, opt.Value)
	}
}

func normalizeOptionKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if !strings.HasPrefix(key, "-") {
		key = "-" + key
	}
	return key
}
