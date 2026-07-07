package main

import (
	"fmt"

	"github.com/Galdoba/ffquery/pkg/media/video/cropdetection"
)

func main() {
	path := `\\192.168.31.4\buffer\IN\anatomiya_padeniya_treyler_2_min.mp4`
	mp, err := cropdetection.GetCropDetectMap(path)
	fmt.Println(err)
	fmt.Println(len(mp))
	fmt.Println(mp[100])

	err = cropdetection.GenerateCSV(mp, path+".crop_detection.csv")
	fmt.Println(err)

}
