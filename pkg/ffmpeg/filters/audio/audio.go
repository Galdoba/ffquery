package audio

import (
	"github.com/Galdoba/ffcmd/ffmpeg/filters/audio/amerge"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/audio/aresample"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/audio/asetnsamples"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/audio/asplit"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/audio/astats"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/audio/atempo"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/audio/bandpass"
	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

func Aresample(opts ...options.Option) *aresample.Aresample { return aresample.New(opts...) }
func Asetnsamples(opts ...options.Option) *asetnsamples.Asetnsamples {
	return asetnsamples.New(opts...)
}
func Atempo(opts ...options.Option) *atempo.Atempo       { return atempo.New(opts...) }
func Asplit(opts ...options.Option) *asplit.Asplit       { return asplit.New(opts...) }
func Amerge(opts ...options.Option) *amerge.Amerge       { return amerge.New(opts...) }
func Astats(opts ...options.Option) *astats.Astats       { return astats.New(opts...) }
func Bandpass(opts ...options.Option) *bandpass.Bandpass { return bandpass.New(opts...) }
