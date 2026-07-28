package yframe_test

import (
	"os"
	"testing"

	"github.com/Galdoba/ffquery/pkg/media/video/blocksize"
	"github.com/Galdoba/ffquery/pkg/media/video/metricfile"
	"github.com/Galdoba/ffquery/pkg/media/video/yframe"
)

func TestScanVideo(t *testing.T) {
	videoPath := `\\192.168.31.4\root\EDIT\@trailers_temp\Adskiy_ray_s01_TRL_HD.mp4`
	outPath := videoPath + ".yblk"
	cfg := blocksize.ConfigExactGridSize(16)
	err := yframe.ScanToFile(videoPath, cfg, outPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify with reader
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	reader, err := metricfile.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	hdr := reader.Header()
	t.Logf("Header: %+v", hdr)
	frame0, err := reader.ReadFrame(0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Frame 0: %v", frame0)
}
