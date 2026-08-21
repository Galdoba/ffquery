package global

import "github.com/Galdoba/ffcmd/ffmpeg/options"

// SetVerbosityLevel – пример специализированного конструктора.
func SetLogLevel(level string) options.Option {
	return options.Opt("-loglevel", level)
}

// OverwriteOutput – опция -y.
func OverwriteOutput() options.Option {
	return options.Flag("-y")
}

// InputFrameRate – опция -r.
func InputFrameRate(fps string) options.Option {
	return options.Opt("-r", fps)
}
