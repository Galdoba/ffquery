package mediagroup

import (
	"fmt"
	"log"
	"testing"
)

func TestNew(t *testing.T) {
	testCMD()
	return
	mg, err := New(`\\192.168.31.4\root\IN\_VESTA\_DONE\Arman\Armand_HD24_DTS.m2ts`)
	fmt.Println(err)
	fmt.Println(mg)
	fmt.Println(mg.MediaFiles[0])
	err = mg.MediaFiles[0].ScanRmsLevels()
	if err != nil {
		log.Fatal(err)
	}

}
