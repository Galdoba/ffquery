package commands

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Galdoba/ffquery/internal/infrastructure"
	"github.com/Galdoba/ffquery/internal/infrastructure/config"
	"github.com/Galdoba/ffquery/pkg/ffprobe"
	"github.com/urfave/cli/v3"
)

func Root() cli.Command {
	cmd := cli.Command{
		Name:           "",
		Usage:          "",
		UsageText:      "",
		ArgsUsage:      "",
		Version:        config.Version,
		Description:    "",
		DefaultCommand: "",
		Category:       "",
		Commands:       []*cli.Command{},
		Flags:          []cli.Flag{},
		Before:         nil,
		After:          nil,
		Action:         rootAction(),
		Authors:        []any{"galdoba"},
	}

	return cmd
}

func rootAction() cli.ActionFunc {
	return func(ctx context.Context, c *cli.Command) error {
		inf := infrastructure.Init()
		logger := inf.GetLogger()
		fmt.Println(inf.GetConfig())
		args := c.Args().Slice()
		for _, arg := range args {
			data, err := ffprobe.NewRawData(arg)
			if err != nil {
				logger.Error("ffprobe failed", "error", err)
				fmt.Println(err)
			} else {
				fmt.Println(data.Render())
				fmt.Println(data)
			}
			hash, err := ffprobe.HashFileWithProgress(arg, 2*time.Second)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("\nSHA-256:", hash)
		}
		return nil
	}
}
