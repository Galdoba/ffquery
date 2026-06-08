package main

import (
	"fmt"

	mediagroup "github.com/Galdoba/ffquery/pkg/media"
)

func main() {
	mg, err := mediagroup.New(`\\192.168.31.4\buffer\IN\_DONE\Terror_s03e05_PRT260605100000_0.4.4_SER_08945_18.mp4`)
	fmt.Println(err)
	fmt.Println(mg.MediaFiles[0].Audio)
	if err := mg.MediaFiles[0].ScanRmsLevels(); err != nil {
		fmt.Println(err)
	}
}
