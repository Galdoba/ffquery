package mediagroup

import (
	"fmt"
	"log"
	"testing"
)

func TestNew(t *testing.T) {
	// testCMD()
	// return
	mg, err := New(`\\192.168.31.4\buffer\IN\_DONE\Terror_s03e05_PRT260605100000_0.4.4_SER_08945_18.mp4`)
	fmt.Println(err)
	fmt.Println(mg)
	fmt.Println(mg.MediaFiles[0])
	err = mg.MediaFiles[0].ScanRmsLevels()
	if err != nil {
		log.Fatal(err)
	}

}
