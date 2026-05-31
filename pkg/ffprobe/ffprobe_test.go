package ffprobe_test

import (
	"fmt"
	"testing"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
)

func TestRawData_ToMedia(t *testing.T) {
	raw, err := ffprobe.NewRawData(`\\192.168.31.4\buffer\IN\_DONE\testing_sources\Dilan_Arbakl_vs_Niko_Leivars_D_PRT260417230822\SPO_40600.mp4`)
	fmt.Println(err)
	// m, err := raw.ToMedia()
	// fmt.Println(err)
	fmt.Println(raw.Format.Filename)
	fmt.Println(raw.Render())
	for _, s := range raw.Streams {
		fmt.Println(s.FPS(), s.Bitrate(), s.DurationTimestamp(), s.Codec(), s.Language(), s.PixFmt, s.SizeFormat())
	}
}
