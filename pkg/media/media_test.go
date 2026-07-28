package mediagroup

import (
	"testing"
)

func TestNew(t *testing.T) {
	// mg, err := New(`\\192.168.31.4\buffer\IN\_DONE\Igra_lzhecov_s01e13_PRT260629200000_0.4.4_SER_08973_18.mp4`)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// media := mg.MediaFiles[0]

	// // Настройка прогресс-бара (опционально)
	// progressOpts := []progress.Option{
	// 	progress.WithOutput(os.Stderr),
	// 	progress.WithTotal(100),
	// 	progress.WithDescription("scanning"),
	// 	progress.WithBarWidth(40),
	// 	progress.WithTemplate(
	// 		progress.KeySpinner + " " + progress.KeyDesc + " " +
	// 			progress.KeyBar + " " + progress.KeyPercent,
	// 	),
	// 	progress.WithSpinner(progress.CommonSpinnerFrames()),
	// 	// progress.WithEndClear(true),
	// }

	// fltrs := []filters.AstatMeasure{filters.RMSLevel, filters.RMSPeak}

	// err = media.ScanAstats(context.Background(),
	// 	WithPerChannelMeasures(fltrs...),
	// 	WithOverallMeasures(fltrs...),
	// 	WithProgressTracker(progressOpts...),
	// )
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("CSV:", media.Meta[astatsCSVKey])
}
