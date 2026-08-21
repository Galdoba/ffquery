package filters

import (
	"fmt"
	"strings"
)

type AstatOption string
type AstatMeasure string

const (
	AstatOptionLength            AstatOption = "length"             //Short window length in seconds, used for peak and through RMS measurement. Default is 0.05 (50 milliseconds). Allowed range is [0 - 10].
	AstatOptionMetadata          AstatOption = "metadata"           //Set metadata injection. All the metadata keys are prefixed with lavfi.astats.X, where X is channel number starting from 1 or string Overall. Default is disabled.
	AstatOptionReset             AstatOption = "reset"              //Set the number of frames over which cumulative stats are calculated before being reset. Default is disabled.
	AstatOptionMeasurePerChannel AstatOption = "measure_perchannel" //Select the parameters which are measured per channel. The metadata keys can be used as flags, default is all which measures everything. none disables all per channel measurement.
	AstatOptionMeasureOverall    AstatOption = "measure_overall"    //Select the parameters which are measured overall. The metadata keys can be used as flags, default is all which measures everything. none disables all overall measurement.

	None              AstatMeasure = "none"                // no measures
	All               AstatMeasure = "all"                 // all measures
	BitDepth          AstatMeasure = "Bit_depth"           // overall bit depth of audio, i.e. number of bits used for each sample
	CrestFactor       AstatMeasure = "Crest_factor"        // standard ratio of peak to RMS level (note: not in dB)
	DCOffset          AstatMeasure = "DC_offset"           // mean amplitude displacement from zero
	DynamicRange      AstatMeasure = "Dynamic_range"       // measured dynamic range of audio in dB
	Entropy           AstatMeasure = "Entropy"             // entropy measured across whole audio, entropy of value near 1.0 is typically measured for white noise
	FlatFactor        AstatMeasure = "Flat_factor"         // flatness (i.e. consecutive samples with the same value) of the signal at its peak levels (i.e. either Min_level or Max_level)
	MaxDifference     AstatMeasure = "Max_difference"      // maximal difference between two consecutive samples
	MaxLevel          AstatMeasure = "Max_level"           // maximal sample level
	MeanDifference    AstatMeasure = "Mean_difference"     // mean difference between two consecutive samples, i.e. the average of each difference between two consecutive samples
	MinDifference     AstatMeasure = "Min_difference"      // minimal difference between two consecutive samples
	MinLevel          AstatMeasure = "Min_level"           // minimal sample level
	NoiseFloor        AstatMeasure = "Noise_floor"         // minimum local peak measured in dBFS over a short window
	NoiseFloorCount   AstatMeasure = "Noise_floor_count"   // number of occasions (not the number of samples) that the signal attained Noise floor
	NumberOfInfs      AstatMeasure = "Number of Infs"      // number of samples with an infinite value
	NumberOfNaNs      AstatMeasure = "Number of NaNs"      // number of samples with a NaN (not a number) value
	NumberOfDenormals AstatMeasure = "Number of denormals" // number of samples with a subnormal value
	NumberOfSamples   AstatMeasure = "Number of samples"   // number of samples
	PeakCount         AstatMeasure = "Peak_count"          // number of occasions (not the number of samples) that the signal attained either Min_level or Max_level
	AbsPeakCount      AstatMeasure = "Abs_Peak_count"      // number of occasions that the absolute samples taken from the signal attained max absolute value of Min_level and Max_level
	PeakLevel         AstatMeasure = "Peak_level"          // standard peak level measured in dBFS
	RMSDifference     AstatMeasure = "RMS_difference"      // Root Mean Square difference between two consecutive samples
	RMSLevel          AstatMeasure = "RMS_level"           // standard RMS level measured in dBFS
	RMSPeak           AstatMeasure = "RMS_peak"            // peak and through values for RMS level measured over a short window, measured in dBFS.
	RMSTrough         AstatMeasure = "RMS_trough"          // peak and through values for RMS level measured over a short window, measured in dBFS.
	ZeroCrossings     AstatMeasure = "Zero_crossings"      // number of points where the waveform crosses the zero level axis
	ZeroCrossingsRate AstatMeasure = "Zero_crossings_rate" // rate of Zero crossings and number of audio samples
)

// Allowed measurements per channel (from FFmpeg documentation).
var allowedPerChannelMeasures = map[string]bool{
	"Bit_depth":           true,
	"Crest_factor":        true,
	"DC_offset":           true,
	"Dynamic_range":       true,
	"Entropy":             true,
	"Flat_factor":         true,
	"Max_difference":      true,
	"Max_level":           true,
	"Mean_difference":     true,
	"Min_difference":      true,
	"Min_level":           true,
	"Noise_floor":         true,
	"Noise_floor_count":   true,
	"Number of Infs":      true,
	"Number of NaNs":      true,
	"Number of denormals": true,
	"Peak_count":          true,
	"Abs_Peak_count":      true,
	"Peak_level":          true,
	"RMS_difference":      true,
	"RMS_level":           true, // WARN: documentation does not sate as allowed, but ffmpeg takes it.
	"RMS_peak":            true,
	"RMS_trough":          true,
	"Zero crossings":      true,
	"Zero crossings rate": true,
}

// Allowed measurements overall (from FFmpeg documentation).
var allowedOverallMeasures = map[string]bool{
	"Bit_depth":           true,
	"DC_offset":           true,
	"Entropy":             true,
	"Flat_factor":         true,
	"Max_difference":      true,
	"Max_level":           true,
	"Mean_difference":     true,
	"Min_difference":      true,
	"Min_level":           true,
	"Noise_floor":         true,
	"Noise_floor_count":   true,
	"Number of Infs":      true,
	"Number of NaNs":      true,
	"Number of denormals": true,
	"Number of samples":   true,
	"Peak_count":          true,
	"Abs_Peak_count":      true,
	"Peak_level":          true,
	"RMS_difference":      true,
	"RMS_level":           true,
	"RMS_peak":            true,
	"RMS_trough":          true,
	"Zero crossings":      true,
	"Zero crossings rate": true,
}

// Astat holds the configuration for the astats audio filter.
type Astat struct {
	Length             float64  // Short window length in seconds (0 – 10).
	Metadata           string   // Metadata injection flag (empty = disabled, otherwise typically "1").
	Reset              int      // Number of frames before cumulative stats are reset (0 = disabled).
	PerChannelMeasures []string // List of per‑channel measures (or ["none"] / ["all"]).
	OverallMeasures    []string // List of overall measures.
}

// AstatOptFunc is a functional option for configuring Astat.
type AstatOptFunc func(*Astat) error

// NewAstat creates a new Astat filter with default values.
func NewAstat(opts ...AstatOptFunc) (*Astat, error) {
	f := &Astat{
		Length: 0.05,
	}
	for _, apply := range opts {
		if err := apply(f); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// MustAstat is same as NewAstat, but panics on error.
func MustAstat(opts ...AstatOptFunc) *Astat {
	as, err := NewAstat(opts...)
	if err != nil {
		panic(err)
	}
	return as
}

// String returns the filter string representation for use in an FFmpeg command.
// Example: astats=metadata=1:reset=1:measure_perchannel=RMS_level+Peak_level:measure_overall=RMS_level+Peak_level
func (a *Astat) String() string {
	parts := make([]string, 0, 4)

	// Length
	if a.Length != 0.05 {
		parts = append(parts, fmt.Sprintf("%s=%v", AstatOptionLength, a.Length))
	}

	// Metadata
	if a.Metadata != "" {
		parts = append(parts, fmt.Sprintf("%s=%s", AstatOptionMetadata, a.Metadata))
	}

	// Reset
	if a.Reset > 0 {
		parts = append(parts, fmt.Sprintf("%s=%d", AstatOptionReset, a.Reset))
	}

	// Per‑channel measures
	if len(a.PerChannelMeasures) > 0 {
		parts = append(parts, fmt.Sprintf("%s=%s", AstatOptionMeasurePerChannel, strings.Join(a.PerChannelMeasures, "+")))
	}

	// Overall measures
	if len(a.OverallMeasures) > 0 {
		parts = append(parts, fmt.Sprintf("%s=%s", AstatOptionMeasureOverall, strings.Join(a.OverallMeasures, "+")))
	}

	if len(parts) == 0 {
		return "astats" // No non‑default options.
	}
	return "astats=" + strings.Join(parts, ":")
}

// AstatLength sets the short window length in seconds (range 0 – 10).
func AstatLength(length float64) AstatOptFunc {
	return func(a *Astat) error {
		if length < 0 || length > 10 {
			return fmt.Errorf("astat length must be in range [0, 10], got %f", length)
		}
		a.Length = length
		return nil
	}
}

// AstatMetadata enables metadata injection.
func AstatMetadata(enable bool) AstatOptFunc {
	return func(a *Astat) error {
		if enable {
			a.Metadata = "1"
		} else {
			a.Metadata = ""
		}
		return nil
	}
}

// AstatReset sets the number of frames over which cumulative stats are calculated before reset.
// A value <= 0 disables the reset.
func AstatReset(frames int) AstatOptFunc {
	return func(a *Astat) error {
		if frames < 0 {
			return fmt.Errorf("astat reset must be non‑negative, got %d", frames)
		}
		a.Reset = frames
		return nil
	}
}

// AstatMeasurePerChannel sets the per‑channel measurements.
// Special values: None (disable all) and All (enable all). Passing None with other keys is an error.
// Each key must be valid for per‑channel measurement (see FFmpeg documentation).
func AstatMeasurePerChannel(measures ...AstatMeasure) AstatOptFunc {
	return func(a *Astat) error {
		keys, err := validateMeasures(measures)
		if err != nil {
			return err
		}
		// "all" and "none" are always valid.
		if len(keys) == 1 && (keys[0] == string(All) || keys[0] == string(None)) {
			a.PerChannelMeasures = keys
			return nil
		}
		for _, k := range keys {
			if !allowedPerChannelMeasures[k] {
				fmt.Printf("measure '%s' is not valid for per‑channel measurement\n", k)
				return nil
				// return fmt.Errorf("measure '%s' is not valid for per‑channel measurement", k)
			}
		}
		a.PerChannelMeasures = keys
		return nil
	}
}

// AstatMeasureOverall sets the overall measurements.
// Special values: None (disable all) and All (enable all). Passing None with other keys is an error.
// Each key must be valid for overall measurement (see FFmpeg documentation).
func AstatMeasureOverall(measures ...AstatMeasure) AstatOptFunc {
	return func(a *Astat) error {
		keys, err := validateMeasures(measures)
		if err != nil {
			return err
		}
		if len(keys) == 1 && (keys[0] == string(All) || keys[0] == string(None)) {
			a.OverallMeasures = keys
			return nil
		}
		for _, k := range keys {
			if !allowedOverallMeasures[k] {
				fmt.Printf("measure '%s' is not valid for overall measurement\n", k)
				return nil
			}
		}
		a.OverallMeasures = keys
		return nil
	}
}

// validateMeasures converts a list of AstatMeasure constants to their string representations,
// enforces the restrictions around None/All, and returns the final string slice.
func validateMeasures(measures []AstatMeasure) ([]string, error) {
	seenNone := false
	seenAll := false
	seenMeasure := make(map[AstatMeasure]bool)
	others := make([]string, 0)

	for _, m := range measures {
		switch m {
		case None:
			seenNone = true
		case All:
			seenAll = true
		default:
			if seenMeasure[m] {
				return nil, fmt.Errorf("cannot call measurement more than once: %v", m)
			}
			others = append(others, string(m))
			seenMeasure[m] = true
		}
	}

	// Error if both None and All are present, or None is combined with specific keys.
	if seenNone && (seenAll || len(others) > 0) {
		return nil, fmt.Errorf("cannot combine 'none' with other measurement keys")
	}

	// If All is used, any other keys are irrelevant; we just output "all".
	if seenAll {
		return []string{string(All)}, nil
	}

	// If None was given alone, output "none".
	if seenNone {
		return []string{string(None)}, nil
	}

	// Only explicit keys.
	return others, nil
}
