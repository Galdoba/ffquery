package progress_test

import (
	"os"
	"testing"
	"time"

	"github.com/Galdoba/ffquery/pkg/progress"
)

func TestTracker(t *testing.T) {
	var total int64 = 1000

	tr := progress.NewTracker(
		progress.WithTotal(total),
		progress.WithOutput(os.Stdout),
		progress.WithSpinner(progress.CommonSpinnerFrames()),
		progress.WithTemplate(
			progress.KeySpinner+" "+progress.KeyDesc+" "+
				progress.KeyBar+" "+progress.KeyPercent+" "+
				progress.KeyCurrent+"/"+progress.KeyTotal+" "+
				progress.KeyElapsed+
				" Speed: "+progress.KeySpeed+" ETA: "+progress.KeyETA+"         ",
		),
		progress.WithDescription("Processing"),
		progress.WithEndClear(false),
		// progress.WithTimeout(10*time.Second), // автоматически закроется через 10 с
	)
	defer tr.Close()

	for i := 0; i <= int(total); i += 1 {
		tr.Set(int64(i))
		time.Sleep(8 * time.Millisecond)
	}
}
