package cropdetection

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// GetCropDetectMap запускает ffmpeg с фильтром cropdetect для указанного видео,
// читает вывод stderr, извлекает строки с "crop=" и возвращает карту,
// где ключ — порядковый номер обнаруженного кадра (начиная с 0),
// а значение — строка с параметрами обрезки (например, "1280:720:0:0").
func GetCropDetectMap(filePath string) (map[int]string, error) {
	// Подготовка команды
	cmd := exec.Command(
		"ffmpeg",
		"-i", filePath,
		"-an", // отключаем аудио
		"-sn", // отключаем субтитры
		"-vf", "cropdetect=limit=16:round=0:reset=1",
		"-f", "null", // выходной формат null, чтобы не создавать файл
		"-", // выходной файл — пустота
	)

	// Получаем stderr команды
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Запускаем команду
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Сканер для построчного чтения stderr
	scanner := bufio.NewScanner(stderr)
	result := make(map[int]string)
	frameIdx := 0

	// Читаем строки, пока не закончатся
	for scanner.Scan() {
		line := scanner.Text()
		// Ищем подстроку "crop="
		if _, after, ok := strings.Cut(line, "crop="); ok {
			// Берём всё после "crop=" до конца строки
			cropStr := after // len("crop=") == 5
			// Обрезаем возможные пробелы/символы в конце
			cropStr = strings.TrimSpace(cropStr)
			if cropStr != "" {
				result[frameIdx] = cropStr
				frameIdx++
			}
		}
		fmt.Printf("processing frame %d\r", frameIdx)
	}
	fmt.Println()

	// Проверяем ошибки сканера
	if err := scanner.Err(); err != nil {
		return result, err
	}

	// Ждём завершения команды и возвращаем ошибку, если она не nil
	if err := cmd.Wait(); err != nil {
		return result, err
	}

	return result, nil
} // }

// GenerateCSV создаёт CSV-файл с данными обрезки из карты cropData.
// Входные данные: map[int]string, где ключ — номер кадра, значение — строка "width:height:x:y".
// Выходной файл сохраняется по пути outputPath.
// Возвращает ошибку, если не удалось создать файл, записать данные или распарсить строку.
func GenerateCSV(cropData map[int]string, outputPath string) error {
	// Создаём файл (перезаписываем, если существует)
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Создаём CSV-писатель
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Заголовок
	header := []string{"frame", "width", "height", "w_offset", "h_offset"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Сортируем ключи (номера кадров) по возрастанию
	keys := make([]int, 0, len(cropData))
	for k := range cropData {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Проходим по каждому кадру в порядке сортировки
	for _, frame := range keys {
		cropStr := cropData[frame]

		// Разбиваем строку "width:height:x:y"
		parts := strings.Split(cropStr, ":")
		if len(parts) != 4 {
			// Если строка некорректна, пропускаем запись (можно также вернуть ошибку)
			continue
		}

		// Преобразуем каждую часть в целое число
		width, err := strconv.Atoi(parts[0])
		if err != nil {
			return err
		}
		height, err := strconv.Atoi(parts[1])
		if err != nil {
			return err
		}
		offsetX, err := strconv.Atoi(parts[2])
		if err != nil {
			return err
		}
		offsetY, err := strconv.Atoi(parts[3])
		if err != nil {
			return err
		}

		// Собираем строку для CSV (все числа как строки)
		record := []string{
			strconv.Itoa(frame),
			strconv.Itoa(width),
			strconv.Itoa(height),
			strconv.Itoa(offsetX),
			strconv.Itoa(offsetY),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
