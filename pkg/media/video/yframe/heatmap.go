package yframe

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/Galdoba/ffquery/pkg/media/video/metricfile"
)

// GenerateHeatmapVideo создаёт видеофайл heatmap из бинарного файла .yblk.
// Каждый блок отображается как 1 пиксель (gray-уровень).
// Если в заголовке нет информации о частоте кадров, используется 30 fps.
func GenerateHeatmapVideo(yblkPath, outputVideoPath string) error {
	// 1. Открываем метрику
	f, err := os.Open(yblkPath)
	if err != nil {
		return fmt.Errorf("open yblk: %w", err)
	}
	defer f.Close()

	reader, err := metricfile.NewReader(f)
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	hdr := reader.Header()

	width := int(hdr.Cols)
	height := int(hdr.Rows)

	// Определяем fps (из заголовка или по умолчанию)
	fpsNum, fpsDen := int(hdr.FPSNum), int(hdr.FPSDen)
	if fpsNum == 0 || fpsDen == 0 {
		fpsNum, fpsDen = 30, 1
	}

	// 2. Готовим ffmpeg: принимает raw gray‑кадры через pipe и кодирует видео
	cmd := exec.Command("ffmpeg",
		"-y", // перезаписать выходной файл
		"-f", "rawvideo",
		"-pixel_format", "gray",
		"-video_size", fmt.Sprintf("%dx%d", width, height),
		"-framerate", fmt.Sprintf("%d/%d", fpsNum, fpsDen),
		"-i", "pipe:0", // читать stdin
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p", // совместимость с плеерами
		outputVideoPath,
	)
	cmd.Stderr = os.Stderr // показывать логи ffmpeg

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	// 3. Последовательно передаём все кадры
	// frameSize := hdr.FrameSize()
	for index := uint64(0); ; index++ {
		frame, err := reader.ReadFrame(index)
		if err != nil {
			// Если достигли конца файла – выходим из цикла
			if err == metricfile.ErrFrameOutOfRange || err == io.EOF {
				break
			}
			stdin.Close()
			cmd.Wait()
			return fmt.Errorf("read frame %d: %w", index, err)
		}

		if _, err := stdin.Write(frame); err != nil {
			// Ошибка записи в пайп (ffmpeg мог упасть)
			break
		}
	}

	// 4. Завершаем ffmpeg
	stdin.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg finished with error: %w", err)
	}
	return nil
}
