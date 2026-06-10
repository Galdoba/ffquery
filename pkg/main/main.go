package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/Galdoba/ffquery/pkg/ffmpeg/filters"
	mediagroup "github.com/Galdoba/ffquery/pkg/media"
)

func main() {
	mg, err := mediagroup.New(`/home/galdoba/Videos/video_1.mp4`)
	fmt.Println(err)
	fmt.Println(mg.MediaFiles[0].Audio)
	if err := mg.MediaFiles[0].ScanAstats(context.Background(), filters.RMSLevel, filters.RMSPeak); err != nil {
		fmt.Println(err)
	}
	fmt.Println(FindMinNumericValue(`/home/galdoba/Videos/video_1.AstatsScan.csv`))
}

// FindMinNumericValue ищет минимальное числовое значение во всем CSV файле.
// Возвращает минимальное значение типа float64 и ошибку, если файл не найден или чисел нет.
func FindMinNumericValue(filePath string) (float64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// FieldsPerRecord = -1 разрешает строкам иметь разное количество колонок,
	// что делает функцию более устойчивой к "битым" CSV файлам.
	reader.FieldsPerRecord = -1

	var minVal float64
	found := false // Флаг, чтобы отличить ситуацию "файл пуст" от "минимум равен 0"

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // Достигнут конец файла
		}
		if err != nil {
			return 0, fmt.Errorf("ошибка чтения строки CSV: %w", err)
		}

		// Проходим по всем ячейкам в текущей строке
		for _, field := range record {
			// Пытаемся преобразовать строку в число с плавающей точкой
			val, parseErr := strconv.ParseFloat(field, 64)

			// Если ошибки нет, значит это число
			if parseErr == nil {
				if !found {
					minVal = val
					found = true
				} else if val < minVal {
					minVal = val
				}
			}
			// Если parseErr != nil, мы просто игнорируем ячейку (это заголовок или текст)
		}
	}

	if !found {
		return 0, errors.New("в файле не найдено ни одного числового значения")
	}

	return minVal, nil
}
