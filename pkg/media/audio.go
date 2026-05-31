package mediagroup

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/Galdoba/ffquery/pkg/ffprobe"
)

// ScanAudioRaw returns an exec.Cmd configured to run a single ffmpeg pass that
// outputs all RMS and LUFS metadata (mix & per‑channel, interval & overall)
// for the requested audio streams to stdout.
//
// Each measurement block is tagged:
//   - rms_int        interval RMS (40 ms blocks)
//   - rms_ovr        overall RMS (accumulated)
//   - mix_lufs       integrated / short‑term LUFS of the whole mix
//   - chX_lufs       integrated / short‑term LUFS of channel X (X = 1,2,…)
//
// The caller should start the command, read its stdout line by line, and
// dispatch the parsed values using the tags.
func (m *Media) ScanAudioRaw(indexes ...int) (*exec.Cmd, error) {
	if m.Raw.Format == nil || m.Raw.Format.Filename == "" {
		return nil, fmt.Errorf("no input file information")
	}

	// Collect all audio streams (global index + audio‑index)
	type audInfo struct {
		globalIndex int
		audioIndex  int
		stream      ffprobe.Stream
	}
	var allAudio []audInfo
	for _, s := range m.Raw.Streams {
		if s.CodecType == ffprobe.StreamTypeAudio {
			allAudio = append(allAudio, audInfo{
				globalIndex: s.Index,
				audioIndex:  len(allAudio),
				stream:      s,
			})
		}
	}
	if len(allAudio) == 0 {
		return nil, fmt.Errorf("no audio streams found")
	}

	// If no indexes given, scan all audio streams
	if len(indexes) == 0 {
		indexes = make([]int, len(allAudio))
		for i, a := range allAudio {
			indexes[i] = a.globalIndex
		}
	}

	// Build the list of streams to process
	var toScan []audInfo
	for _, idx := range indexes {
		found := false
		for _, a := range allAudio {
			if a.globalIndex == idx {
				toScan = append(toScan, a)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("audio stream index %d not found or not audio", idx)
		}
	}

	// Determine video frame rate for RMS interval (fallback 25 fps)
	videoFPS := 25.0
	for _, s := range m.Raw.Streams {
		if s.CodecType == ffprobe.StreamTypeVideo {
			if fps := s.FPS(); fps > 0 {
				videoFPS = fps
				break
			}
		}
	}

	// Platform‑specific null device
	nullDev := "/dev/null"
	if runtime.GOOS == "windows" {
		nullDev = "NUL"
	}

	var (
		filterParts []string
		// mapLabels   []string
	)

	for _, aud := range toScan {
		s := aud.stream
		globalIdx := aud.globalIndex
		audioIdx := aud.audioIndex

		sampleRate := s.SampleRateHz()
		if sampleRate <= 0 {
			return nil, fmt.Errorf("stream %d: invalid sample rate", globalIdx)
		}
		channels := s.Channels
		if channels <= 0 {
			return nil, fmt.Errorf("stream %d: invalid channel count", globalIdx)
		}
		layout := s.ChannelLayout
		if layout == "" {
			switch channels {
			case 1:
				layout = "mono"
			case 2:
				layout = "stereo"
			case 6:
				layout = "5.1"
			case 8:
				layout = "7.1"
			default:
				return nil, fmt.Errorf("stream %d: unsupported channel layout (%d ch)", globalIdx, channels)
			}
		}

		samplesPerFrame := int(math.Round(float64(sampleRate) / videoFPS))
		if samplesPerFrame < 1 {
			samplesPerFrame = 1
		}

		// Уникальные метки
		lblMix := fmt.Sprintf("a_lufs_mix_%d", globalIdx)
		lblRmsInt := fmt.Sprintf("a_rms_int_%d", globalIdx)
		lblRmsOvr := fmt.Sprintf("a_rms_ovr_%d", globalIdx)
		lblChSplit := fmt.Sprintf("a_lufs_ch_%d", globalIdx)

		// asplit
		splitLine := fmt.Sprintf("[0:a:%d]asplit=4 [%s] [%s] [%s] [%s]",
			audioIdx, lblMix, lblRmsInt, lblRmsOvr, lblChSplit)
		filterParts = append(filterParts, splitLine)

		// --- LUFS mix (тег mix_lufs) ---
		mixLine := fmt.Sprintf(
			"[%s] ebur128=video=0:meter=18:metadata=1, ametadata=mode=add:key=tag:value=mix_lufs, ametadata=mode=print:file=-, anullsink",
			lblMix)
		filterParts = append(filterParts, mixLine)

		// --- RMS intervals (тег rms_int) ---
		rmsIntLine := fmt.Sprintf(
			"[%s] asetnsamples=%d, astats=metadata=1:reset=1, ametadata=mode=add:key=tag:value=rms_int, ametadata=mode=print:file=-, anullsink",
			lblRmsInt, samplesPerFrame)
		filterParts = append(filterParts, rmsIntLine)

		// --- RMS overall (тег rms_ovr) ---
		rmsOvrLine := fmt.Sprintf(
			"[%s] astats=metadata=1:reset=0, ametadata=mode=add:key=tag:value=rms_ovr, ametadata=mode=print:file=-, anullsink",
			lblRmsOvr)
		filterParts = append(filterParts, rmsOvrLine)

		// --- Per‑channel LUFS ---
		chLabels := make([]string, channels)
		for ch := 0; ch < channels; ch++ {
			chLabels[ch] = fmt.Sprintf("ch%d_%d", ch+1, globalIdx)
		}

		chSplitLine := fmt.Sprintf("[%s] channelsplit=channel_layout=%s %s",
			lblChSplit, layout,
			strings.Join(bracketize(chLabels), " "))
		filterParts = append(filterParts, chSplitLine)

		for ch := 0; ch < channels; ch++ {
			tagVal := fmt.Sprintf("ch%d_lufs", ch+1)
			chLine := fmt.Sprintf(
				"[%s] ebur128=video=0:meter=18:metadata=1, ametadata=mode=add:key=tag:value=%s, ametadata=mode=print:file=-, anullsink",
				chLabels[ch], tagVal)
			filterParts = append(filterParts, chLine)
		}
	}

	filterGraph := strings.Join(filterParts, "; ")
	args := []string{
		"-i", m.Raw.Format.Filename,
		"-filter_complex", filterGraph,
		"-f", "null", nullDev,
	}
	return exec.Command("ffmpeg", args...), nil
}

// bracketize wraps each string in square brackets.
func bracketize(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = "[" + v + "]"
	}
	return out
}

// func Run() {
// 	mg, err := New(`\\192.168.31.4\buffer\IN\_DONE\testing_sources\Aston_Braun_vs_Sem_Dzhilli_D_PRT260418001456\test.mov`)
// 	fmt.Println(err)
// 	fmt.Println(mg)
// 	fmt.Println(mg.MediaFiles[0])
// 	cmd, err := mg.MediaFiles[0].ScanAudioRaw()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// stdout направляем в pipe для парсинга
// 	// stdout, err := cmd.StdoutPipe()
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	stdout, _ := cmd.StdoutPipe()
// 	cmd.Stderr = os.Stderr // прогресс-бар виден сразу
// 	cmd.Start()

// 	scanner := bufio.NewScanner(stdout)
// 	scanner.Buffer(make([]byte, 1<<20), 10<<20)

// 	var currentTag, frameTime string

// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		// fmt.Fprintln(os.Stderr, line) // диагностический вывод (можно удалить)

// 		if strings.HasPrefix(line, "frame:") {
// 			for _, f := range strings.Fields(line) {
// 				if strings.HasPrefix(f, "pts_time=") {
// 					frameTime = strings.TrimPrefix(f, "pts_time=")
// 					break
// 				}
// 			}
// 			continue
// 		}
// 		if strings.HasPrefix(line, "tag=") {
// 			currentTag = strings.TrimPrefix(line, "tag=")
// 			continue
// 		}

// 		switch currentTag {
// 		case "rms_int", "rms_ovr":
// 			if strings.Contains(line, "RMS_level") || strings.Contains(line, "Peak_level") ||
// 				strings.Contains(line, "RMS_peak") || strings.Contains(line, "RMS_trough") {
// 				// сохранение
// 				fmt.Fprintln(os.Stderr, "RMS: "+line) // диагностический вывод (можно удалить)

// 			}
// 		case "mix_lufs", "ch1_lufs", "ch2_lufs":
// 			if strings.Contains(line, "lavfi.r128.M") || strings.Contains(line, "lavfi.r128.S") ||
// 				strings.Contains(line, "lavfi.r128.I") {
// 				fmt.Fprintln(os.Stderr, "LUFS: "+line) // диагностический вывод (можно удалить)
// 				// сохранение
// 			}
// 		}
// 	}
// 	if err := scanner.Err(); err != nil {
// 		log.Fatal("scanner error:", err)
// 	}
// 	cmd.Wait()
// 	frameTime += ""
// }

func Run() {
	mg, err := New(`\\192.168.31.4\buffer\IN\_DONE\testing_sources\Aston_Braun_vs_Sem_Dzhilli_D_PRT260418001456\test.mov`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(err)
	fmt.Println(mg)
	fmt.Println(mg.MediaFiles[0])

	// Получаем исходную команду ffmpeg
	cmd, err := mg.MediaFiles[0].ScanAudioRaw()
	if err != nil {
		log.Fatal(err)
	}

	// Запускаем команду с отдельными pipe для stdout и stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Объединяем stdout и stderr в один io.Reader через io.Pipe
	mergedReader, mergedWriter := io.Pipe()

	var wg sync.WaitGroup
	wg.Add(2)

	// Горутина для копирования stdout в mergedWriter
	go func() {
		defer wg.Done()
		io.Copy(mergedWriter, stdoutPipe)
	}()
	// Горутина для копирования stderr в mergedWriter
	go func() {
		defer wg.Done()
		io.Copy(mergedWriter, stderrPipe)
	}()
	// Когда оба потока закончатся, закрываем писатель mergedWriter
	go func() {
		wg.Wait()
		mergedWriter.Close()
	}()

	// Сканер читает объединённый поток
	scanner := bufio.NewScanner(mergedReader)
	scanner.Buffer(make([]byte, 1<<20), 10<<20)

	var currentTag string
	var frameTime string

	for scanner.Scan() {
		line := scanner.Text()

		// Диагностика: можно вывести всё, что идёт (чтобы убедиться, что потоки объединены)
		// fmt.Fprintln(os.Stderr, "DEBUG:", line)

		// Парсим строки метаданных
		if strings.HasPrefix(line, "frame:") {
			for _, f := range strings.Fields(line) {
				if strings.HasPrefix(f, "pts_time=") {
					frameTime = strings.TrimPrefix(f, "pts_time=")
					break
				}
			}
			continue
		}
		if strings.HasPrefix(line, "tag=") {
			currentTag = strings.TrimPrefix(line, "tag=")
			continue
		}

		// Фильтрация и вывод только нужных данных
		var shouldPrint bool
		switch currentTag {
		case "rms_int", "rms_ovr":
			if strings.Contains(line, "RMS_level") || strings.Contains(line, "Peak_level") ||
				strings.Contains(line, "RMS_peak") || strings.Contains(line, "RMS_trough") {
				shouldPrint = true
				line = "RMS: " + line
			}
		case "mix_lufs", "ch1_lufs", "ch2_lufs":
			if strings.Contains(line, "lavfi.r128.M") || strings.Contains(line, "lavfi.r128.S") ||
				strings.Contains(line, "lavfi.r128.I") {
				line = "LUFS: " + line
				shouldPrint = true
			}
		}
		if strings.Contains(line, "time=") {
			shouldPrint = true
			line = "PROGRESS: " + line
		}
		if shouldPrint {
			fmt.Println(line) // только отфильтрованное выводится в консоль
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("scanner error: %v", err)
	}

	// Дожидаемся завершения ffmpeg
	if err := cmd.Wait(); err != nil {
		log.Printf("ffmpeg exited with error: %v", err)
	}
	fmt.Println("Finished, last frame time:", frameTime)
}
