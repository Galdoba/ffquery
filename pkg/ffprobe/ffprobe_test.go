package ffprobe_test

import (
	"fmt"
	"testing"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
)

func TestRawData_ToMedia(t *testing.T) {
	raw, err := ffprobe.NewRawData(`\\192.168.31.4\buffer\IN\_DONE\Grab_nagrablennoe_iskuplenie_s02e01_PRT260522003403_0.4.4_SER_08869_18.mp4`)
	fmt.Println(err)
	// m, err := raw.ToMedia()
	// fmt.Println(err)
	// fmt.Println(m)
	fmt.Println(raw.Render())
	raw, err = ffprobe.NewRawData(`\\192.168.31.4\buffer\IN\_DONE\Grab_nagrablennoe_iskuplenie_s02e02_PRT260522004758_0.4.4_SER_08870_18.mp4`)
	fmt.Println(err)
	// m, err := raw.ToMedia()
	// fmt.Println(err)
	// fmt.Println(m)
	fmt.Println(raw.Render())
}
