package mediagroup

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
)

func TestNew(t *testing.T) {
	// testCMD()
	// return
	mg, err := New(`\\192.168.31.4\buffer\IN\_DONE\Terror_s03e05_PRT260605100000_0.4.4_SER_08945_18.mp4`)
	fmt.Println(err)
	fmt.Println(mg)
	fmt.Println(mg.MediaFiles[0])
	fltrs := []filters.AstatMeasure{filters.RMSLevel, filters.RMSPeak}
	err = mg.MediaFiles[0].ScanAstats(context.Background(), fltrs, fltrs)
	if err != nil {
		log.Fatal(err)
	}

}
