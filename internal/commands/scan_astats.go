package commands

import (
	"context"
	"fmt"
	"strings"

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
		// Flags: []cli.Flag{
		// 	flags.AstatsMeasurments,
		// },
		Before:  nil,
		After:   nil,
		Action:  scanAstatsAction(),
		Authors: []any{"galdoba"},
	}

	return &cmd
}

func scanAstatsAction() cli.ActionFunc {
	return func(ctx context.Context, c *cli.Command) error {
		inf := infrastructure.Init()
		logger := inf.GetLogger()
		errs := []error{}
		// fmt.Printf("%v\n", c.StringSlice(scanastats.FlagMeasurmentsPerchannel))
		// return nil
		mg, err := mediagroup.New(c.Args().Slice()...)
		if err != nil {
			return fmt.Errorf("failed to create mediagroup: %w", err)
		}
		for _, m := range mg.MediaFiles {
			logger.Info("start scan", "file", m.Path)
			defaultMeasurements := []filters.AstatMeasure{filters.RMSLevel, filters.PeakLevel}
			if err := m.ScanAstats(context.Background(), defaultMeasurements, defaultMeasurements); err != nil {
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
