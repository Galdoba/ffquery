package config

import "github.com/Galdoba/appcontext/xdg"

func Default() Config {
	return Config{
		Persistence: Persistence{
			JSON: JsonStore{
				Path:   defaultJsonStoragePath(),
				Prefix: "",
				Indent: "  ",
			},
		},
		Logger: Logger{
			Path:             defaultLogPath(),
			FileSizeRotation: false,
			SizeLimitMB:      0,
			PeriodicRotation: false,
			Period:           NoPeriod,
		},
	}
}

func defaultJsonStoragePath() string {
	return xdg.Location(xdg.ForData(), xdg.WithProgramName(AppName), xdg.WithFileName("storage.json"))
}

func defaultLogPath() string {
	return xdg.Location(xdg.ForState(), xdg.WithProgramName(AppName), xdg.WithFileName(AppName+".log"))

}
