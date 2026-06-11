package scanastats

import (
	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
	"github.com/urfave/cli/v3"
)

const (
	FlagMeasurmentsPerchannel = "measurements_per_channel"
)

var MeasurmentsCombined = &cli.StringSliceFlag{
	Name:        FlagMeasurmentsPerchannel,
	Category:    "",
	DefaultText: `"RMS_level,Peak_level"`,
	HideDefault: false,
	Aliases:     []string{"mp"},
	Usage:       "set measurements applied to both per_channel and overall metrics",
	Value:       []string{string(filters.RMSLevel), string(filters.PeakLevel)},
}
