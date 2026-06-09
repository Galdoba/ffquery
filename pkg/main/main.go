package main

import (
	"fmt"

	mediagroup "github.com/Galdoba/ffquery/pkg/media"
)

func main() {
	mg, err := mediagroup.New(`\\192.168.31.4\root\IN\@TRAILERS\_DONE\Agent_nacionalnoy_bezopasnosti_vozvrashenie_s01_TRL\agent_nb_treyler_wink.mp4`)
	fmt.Println(err)
	fmt.Println(mg.MediaFiles[0].Audio)
	if err := mg.MediaFiles[0].ScanRmsLevels(); err != nil {
		fmt.Println(err)
	}
}
