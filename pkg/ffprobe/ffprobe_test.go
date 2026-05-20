package ffprobe_test

import (
	"fmt"
	"testing"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
)

func TestRawData_ToMedia(t *testing.T) {
	raw, err := ffprobe.NewRawData(`\\192.168.31.4\root\IN\_AMEDIA\_DONE\Adskiy_ray_s01\Adskiy_ray_s01e09_PRT260108000659_SER_08191_18.mp4`)
	fmt.Println(err)
	m, err := raw.ToMedia()
	fmt.Println(err)
	fmt.Println(m)
	fmt.Println(m.Render(ffprobe.FieldWidth, ffprobe.FieldHeight, ffprobe.FieldLanguage, ffprobe.FieldAvgFrameRate, ffprobe.FieldBitRate, ffprobe.FieldCodec, ffprobe.FieldChannelLayout, ffprobe.FieldName))
}
