package scanastats

import (
	"github.com/urfave/cli/v3"
)

const (
	FlagMeasurmentsCombined   = "measurements"
	FlagMeasurmentsPerchannel = "measurements_per_channel"
	FlagMeasurmentsOverall    = "measurements_overall"
)

var MeasurmentsCombined = &cli.StringFlag{
	Name:        FlagMeasurmentsCombined,
	Category:    "",
	DefaultText: ``,
	HideDefault: false,
	Aliases:     []string{"mc"},
	Usage:       "\nset measurements applied to both per_channel and overall metrics. Excludes other measerements flags.",
}
var MeasurmentsPerChannel = &cli.StringFlag{
	Name:        FlagMeasurmentsPerchannel,
	Category:    "",
	DefaultText: `"RMS_level,Min_level,Max_level,DC_offset,Noise_floor,Max_difference"`,
	HideDefault: false,
	Aliases:     []string{"mp"},
	Usage:       "set measurements applied to per_channel metrics",
	Value:       `RMS_level,Min_level,Max_level,DC_offset,Noise_floor,Max_difference`,
}
var MeasurmentsOverall = &cli.StringFlag{
	Name:        FlagMeasurmentsOverall,
	Category:    "",
	DefaultText: `"RMS_level,Min_level,Max_level,DC_offset,Noise_floor,Max_difference"`,
	HideDefault: false,
	Aliases:     []string{"mo"},
	Usage:       "set measurements applied to overall metrics",
	Value:       "RMS_level,Min_level,Max_level,DC_offset,Noise_floor,Max_difference",
}

// string(filters.RMSLevel),
// string(filters.MinLevel),
// string(filters.MaxLevel),
// string(filters.PeakLevel),
// string(filters.RMSPeak),
// string(filters.RMSTrough),
// string(filters.DCOffset),
// string(filters.NoiseFloor),
// string(filters.NoiseFloorCount),
// string(filters.Entropy),
// string(filters.FlatFactor),
// string(filters.MinDifference),
// string(filters.MeanDifference),
// string(filters.MaxDifference),
// string(filters.RMSDifference),
// string(filters.ZeroCrossings),
// string(filters.ZeroCrossingsRate),
// string(filters.CrestFactor),
// string(filters.DynamicRange),
// string(filters.PeakCount),
// string(filters.AbsPeakCount),
// string(filters.NumberOfSamples),
// string(filters.BitDepth),
