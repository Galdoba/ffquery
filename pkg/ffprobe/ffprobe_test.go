package ffprobe_test

import (
	"fmt"
	"testing"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
)

func TestRawData_ToMedia(t *testing.T) {
	raw, err := ffprobe.NewRawData(`\\192.168.31.4\buffer\IN\_DONE\Slaym_ochen_lipkoe_priklyuchenie_de_nog_grotere_slijmfilm--FILM--DeNogGrotereSlijmfilm_FTR_en_nl20_nl51_en20_en51_1920x1080_25.mov`)
	fmt.Println(err)
	// m, err := raw.ToMedia()
	// fmt.Println(err)
	fmt.Println(raw.Format.Filename)
	fmt.Println(raw.Render())
	for _, s := range raw.Streams {
		fmt.Println(s.FPS(), s.Bitrate(), s.DurationTimestamp(), s.Codec(), s.Language(), s.PixFmt, s.SizeFormat())
	}
}
