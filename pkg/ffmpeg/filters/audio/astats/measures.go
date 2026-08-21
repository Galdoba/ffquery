package astats

import (
	"fmt"
	"strings"
)

// Measure — строковый тип для ключа измерения в фильтре astats.
type Measure string

// Константы всех возможных мер (из документации FFmpeg).
const (
	MeasureBitDepth           Measure = "Bit_depth"
	MeasureCrestFactor        Measure = "Crest_factor"
	MeasureDCOffset           Measure = "DC_offset"
	MeasureDynamicRange       Measure = "Dynamic_range"
	MeasureEntropy            Measure = "Entropy"
	MeasureFlatFactor         Measure = "Flat_factor"
	MeasureMaxDifference      Measure = "Max_difference"
	MeasureMaxLevel           Measure = "Max_level"
	MeasureMeanDifference     Measure = "Mean_difference"
	MeasureMinDifference      Measure = "Min_difference"
	MeasureMinLevel           Measure = "Min_level"
	MeasureNoiseFloor         Measure = "Noise_floor"
	MeasureNoiseFloorCount    Measure = "Noise_floor_count"
	MeasureNumberOfInfs       Measure = "Number_of_Infs"
	MeasureNumberOfNaNs       Measure = "Number_of_NaNs"
	MeasureNumberOfDenormals  Measure = "Number_of_denormals"
	MeasureNumberOfSamples    Measure = "Number_of_samples"
	MeasurePeakCount          Measure = "Peak_count"
	MeasureAbsPeakCount       Measure = "Abs_Peak_count"
	MeasurePeakLevel          Measure = "Peak_level"
	MeasureRMSDifference      Measure = "RMS_difference"
	MeasureRMSLevel           Measure = "RMS_level"
	MeasureRMSPeak            Measure = "RMS_peak"
	MeasureRMSTrough          Measure = "RMS_trough"
	MeasureZeroCrossings      Measure = "Zero_crossings"
	MeasureZeroCrossingsRate  Measure = "Zero_crossings_rate"
)

// PerChannelMeasures — набор мер для опции measure_perchannel.
// Допустимые меры: все, кроме MeasureNumberOfSamples.
type PerChannelMeasures struct {
	keys []Measure
	err  error
}

// NewPerChannelMeasures создаёт набор мер для per-channel.
// Можно передать специальные значения: "none" или "all" (в виде Measure("none") / Measure("all")).
// При ошибке (недопустимая мера) сохраняет ошибку.
func NewPerChannelMeasures(measures ...Measure) *PerChannelMeasures {
	m := &PerChannelMeasures{}
	for _, measure := range measures {
		if err := m.add(measure); err != nil {
			m.err = err
			break
		}
	}
	if m.err == nil && len(m.keys) == 0 {
		m.keys = []Measure{Measure("all")} // по умолчанию all
	}
	return m
}

func (m *PerChannelMeasures) add(measure Measure) error {
	if measure == "none" || measure == "all" {
		if len(m.keys) > 0 {
			return fmt.Errorf("cannot combine 'none' or 'all' with other measures")
		}
		m.keys = append(m.keys, measure)
		return nil
	}
	// Проверяем, что мера допустима для per-channel
	if measure == MeasureNumberOfSamples {
		return fmt.Errorf("measure '%s' is not valid for per-channel", measure)
	}
	if _, ok := allowedPerChannel[string(measure)]; !ok {
		return fmt.Errorf("measure '%s' is not valid for per-channel", measure)
	}
	m.keys = append(m.keys, measure)
	return nil
}

// String возвращает строку для опции measure_perchannel (например, "Peak_level+RMS_level").
func (m *PerChannelMeasures) String() string {
	if m.err != nil {
		return ""
	}
	parts := make([]string, len(m.keys))
	for i, k := range m.keys {
		parts[i] = string(k)
	}
	return strings.Join(parts, "+")
}

// Err возвращает ошибку, если она была при создании набора.
func (m *PerChannelMeasures) Err() error { return m.err }

// OverallMeasures — набор мер для опции measure_overall.
type OverallMeasures struct {
	keys []Measure
	err  error
}

// NewOverallMeasures создаёт набор мер для overall.
// Можно передать специальные значения: "none" или "all".
func NewOverallMeasures(measures ...Measure) *OverallMeasures {
	m := &OverallMeasures{}
	for _, measure := range measures {
		if err := m.add(measure); err != nil {
			m.err = err
			break
		}
	}
	if m.err == nil && len(m.keys) == 0 {
		m.keys = []Measure{Measure("all")} // по умолчанию all
	}
	return m
}

func (m *OverallMeasures) add(measure Measure) error {
	if measure == "none" || measure == "all" {
		if len(m.keys) > 0 {
			return fmt.Errorf("cannot combine 'none' or 'all' with other measures")
		}
		m.keys = append(m.keys, measure)
		return nil
	}
	if _, ok := allowedOverall[string(measure)]; !ok {
		return fmt.Errorf("measure '%s' is not valid for overall", measure)
	}
	m.keys = append(m.keys, measure)
	return nil
}

// String возвращает строку для опции measure_overall.
func (m *OverallMeasures) String() string {
	if m.err != nil {
		return ""
	}
	parts := make([]string, len(m.keys))
	for i, k := range m.keys {
		parts[i] = string(k)
	}
	return strings.Join(parts, "+")
}

// Err возвращает ошибку.
func (m *OverallMeasures) Err() error { return m.err }