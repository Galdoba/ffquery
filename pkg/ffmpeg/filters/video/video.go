package video

import (
	"github.com/Galdoba/ffcmd/ffmpeg/filters/video/crop"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/video/cropdetect"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/video/format"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/video/pad"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/video/scale"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/video/setsar"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/video/split"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/video/subtitles"
	"github.com/Galdoba/ffcmd/ffmpeg/filters/video/unsharp"
	"github.com/Galdoba/ffcmd/ffmpeg/options"
)

func Scale(opts ...options.Option) *scale.Scale                { return scale.New(opts...) }
func SetSar(opts ...options.Option) *setsar.SetSar             { return setsar.New(opts...) }
func Unsharp(opts ...options.Option) *unsharp.Unsharp          { return unsharp.New(opts...) }
func Pad(opts ...options.Option) *pad.Pad                      { return pad.New(opts...) }
func Crop(opts ...options.Option) *crop.Crop                   { return crop.New(opts...) }
func Cropdetect(opts ...options.Option) *cropdetect.Cropdetect { return cropdetect.New(opts...) }
func Format(opts ...options.Option) *format.Format             { return format.New(opts...) }
func Split(opts ...options.Option) *split.Split                { return split.New(opts...) }
func Subtitles(opts ...options.Option) *subtitles.Subtitles    { return subtitles.New(opts...) }
