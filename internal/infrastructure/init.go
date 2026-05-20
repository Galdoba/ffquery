package infrastructure

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Galdoba/appcontext/configmanager"
	"github.com/Galdoba/ffquery/internal/infrastructure/config"
)

type infrastructure struct {
	Cfg    config.Config
	Logger *slog.Logger
}

type Infrastructure interface {
	GetConfig() config.Config
	GetLogger() *slog.Logger
}

func Init() Infrastructure {
	inf, err := New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "panic: failed to create infrastructure: %v\n", err)
		os.Exit(1)
	}
	return inf
}

func New() (Infrastructure, error) {
	inf := infrastructure{}
	cfg, err := configmanager.LazyInit(config.AppName, config.Default())
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.Logger.Path == "" {
		return nil, fmt.Errorf("logger path is not set")
	}
	logpath := cfg.Logger.Path
	inf.Cfg = cfg
	os.MkdirAll(filepath.Dir(logpath), 0777)
	f, err := os.OpenFile(logpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("cannot open logfile %q: %w", logpath, err)
	}

	logger := slog.New(
		slog.NewJSONHandler(f, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	inf.Logger = logger
	return &inf, nil
}

func (inf *infrastructure) GetConfig() config.Config {
	return inf.Cfg
}

func (inf *infrastructure) GetLogger() *slog.Logger {
	return inf.Logger
}
