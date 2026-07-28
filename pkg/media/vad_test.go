package mediagroup_test

import (
	"fmt"
	"testing"

	"github.com/Galdoba/ffquery/pkg/media"
)

func TestDetectSpeechSegments(t *testing.T) {
	csvPath := `\\192.168.31.4\buffer\IN\test1_DanielsGottaDie_HD25f_RUS20LR_RUS51LRCLfeLsRs_NoP.AstatsScan.csv`
	cfg := mediagroup.DefaultVADConfig()
	// Для более строгой сегментации можно подкрутить:
	// cfg.ScoreOn = 0.8
	// cfg.ScoreOff = 0.5
	// cfg.MaxFramesOff = 1
	segments, err := mediagroup.DetectSpeechSegments(csvPath, cfg)

	fmt.Println(err)
	for i, s := range segments {
		fmt.Println(i, s)
	}
}
