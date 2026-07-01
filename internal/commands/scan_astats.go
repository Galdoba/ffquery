package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	scanastats "github.com/Galdoba/ffquery/internal/commands/scan_astats"
	"github.com/Galdoba/ffquery/internal/infrastructure"
	"github.com/Galdoba/ffquery/internal/infrastructure/config"
	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
	mediagroup "github.com/Galdoba/ffquery/pkg/media"
	"github.com/Galdoba/ffquery/pkg/progress"
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
			if err := m.ScanAstats(context.Background(),
				mediagroup.WithPerChannelMeasures(perChannelMeasurments...),
				mediagroup.WithOverallMeasures(overallMeasurments...),
				mediagroup.WithProgressTracker(
					progress.WithTemplate(fmt.Sprintf("scanning: %s %s  elapsed: %s", progress.KeyBar, progress.KeyPercent, progress.KeyElapsed)),
					progress.WithOutput(os.Stderr),
					progress.WithTimeFormatter(hhmmss),
				),
			); err != nil {
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

func hhmmss(d time.Duration) string {
	sign := ""
	if d < 0 {
		sign = "-"
		d = -d
	}
	totalSec := int64(d / time.Second)

	hours := totalSec / 3600
	minutes := (totalSec % 3600) / 60
	seconds := totalSec % 60

	return fmt.Sprintf("%s%02d:%02d:%02d", sign, hours, minutes, seconds)
}
