package commands

import (
	"context"
	"fmt"
	"strings"

	scanastats "github.com/Galdoba/ffquery/internal/commands/scan_astats"
	"github.com/Galdoba/ffquery/internal/infrastructure"
	"github.com/Galdoba/ffquery/internal/infrastructure/config"
	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
	mediagroup "github.com/Galdoba/ffquery/pkg/media"
	"github.com/urfave/cli/v3"
)

func ScanAstats() *cli.Command {
	cmd := cli.Command{
		Name:           "scan-astats",
		Usage:          "scan audio streams with ffmpeg astats filter",
		UsageText:      "",
		ArgsUsage:      "",
		Version:        config.Version,
		Description:    "Generate and run ffmpeg command with selected astats measurements. After that parse results file and compose a csv with measurements data.",
		DefaultCommand: "",
		Category:       "",
		Commands:       []*cli.Command{},
		Flags: []cli.Flag{
			scanastats.MeasurmentsCombined,
			scanastats.MeasurmentsPerChannel,
			scanastats.MeasurmentsOverall,
		},
		Before:  nil,
		After:   nil,
		Action:  scanAstatsAction(),
		Authors: []any{"galdoba"},
	}

	return &cmd
}

var per_channel_metrics = []filters.AstatMeasure{
	filters.RMSLevel,
	filters.RMSPeak,
	filters.RMSTrough,
	filters.RMSDifference,
	filters.MinLevel,
	filters.PeakLevel,
	filters.NoiseFloor,
	filters.NoiseFloorCount,
	filters.DCOffset,
	filters.Entropy,
	filters.FlatFactor,
	filters.MaxDifference,
	filters.MeanDifference,
	filters.ZeroCrossings,
	filters.ZeroCrossingsRate,
	filters.BitDepth,
}

var overall_metrics = []filters.AstatMeasure{
	filters.RMSLevel,
	filters.RMSPeak,
	filters.RMSTrough,
	filters.RMSDifference,
	filters.MinLevel,
	filters.PeakLevel,
	filters.NoiseFloor,
	filters.NoiseFloorCount,
	filters.DCOffset,
	filters.Entropy,
	filters.FlatFactor,
	filters.MaxDifference,
	filters.MeanDifference,
	filters.BitDepth,
}

func scanAstatsAction() cli.ActionFunc {
	return func(ctx context.Context, c *cli.Command) error {
		inf := infrastructure.Init()
		logger := inf.GetLogger()
		errs := []error{}
		fmt.Printf("%v\n", c.String(scanastats.FlagMeasurmentsPerchannel))
		// return nil
		mg, err := mediagroup.New(c.Args().Slice()...)
		if err != nil {
			return fmt.Errorf("failed to create mediagroup: %w", err)
		}
		perChannelMeasurments := []filters.AstatMeasure{}
		overallMeasurments := []filters.AstatMeasure{}
		switch c.String(scanastats.FlagMeasurmentsCombined) {
		case "":
			for s := range strings.SplitSeq(c.String(scanastats.FlagMeasurmentsPerchannel), ",") {
				perChannelMeasurments = append(perChannelMeasurments, filters.AstatMeasure(s))
			}
			for s := range strings.SplitSeq(c.String(scanastats.FlagMeasurmentsOverall), ",") {
				overallMeasurments = append(overallMeasurments, filters.AstatMeasure(s))
			}
		default:
			for s := range strings.SplitSeq(c.String(scanastats.FlagMeasurmentsCombined), ",") {
				perChannelMeasurments = append(perChannelMeasurments, filters.AstatMeasure(s))
				overallMeasurments = append(overallMeasurments, filters.AstatMeasure(s))
			}
		}

		for _, m := range mg.MediaFiles {
			logger.Info("start scan", "file", m.Path)
			if err := m.ScanAstats(context.Background(), perChannelMeasurments, overallMeasurments); err != nil {
				errs = append(errs, fmt.Errorf("failed to scan file %s: %w", m.Path, err))
				continue
			}
			logger.Error("scan completed", "result file", m.Meta["astatsCSV"])
		}
		if len(errs) > 0 {
			return errorReport(errs...)
		}
		return nil
	}
}

func errorReport(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}
	var s strings.Builder
	fmt.Fprintf(&s, "%d errors detected:\n", len(errs))
	for _, err := range errs {
		fmt.Fprintf(&s, "  %v\n", err)
	}
	return fmt.Errorf("%s", s.String())
}
