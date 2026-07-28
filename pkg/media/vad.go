package mediagroup

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/Galdoba/ffquery/pkg/media/streammap"
)

// SpeechSegment представляет непрерывный участок речи в одном канале конкретного аудиопотока.
type SpeechSegment struct {
	Stream     int
	Channel    int
	StartFrame int
	EndFrame   int
	StartSec   float64
	EndSec     float64
	Confidence float64
}

// VADConfig содержит параметры детектора речи.
type VADConfig struct {
	SNRThreshold         float64
	CrestTypical         float64
	TransientThr         float64
	SpeechRatioThreshold float64
	W1, W2, W3, W4       float64
	ScoreOn              float64
	ScoreOff             float64
	MinFramesOn          int
	MaxFramesOff         int

	// ИСПРАВЛЕНИЕ: масштабные коэффициенты для нормализации признаков
	SNRScale         float64 // типичный разброс SNR (дБ)
	CrestScale       float64 // разброс пик-фактора (дБ)
	SpeechRatioScale float64 // разброс speech ratio (дБ)
}

func DefaultVADConfig() VADConfig {
	return VADConfig{
		SNRThreshold:         6.0,
		CrestTypical:         12.0,
		TransientThr:         -20.0,
		SpeechRatioThreshold: -6.0,
		W1:                   0.4, W2: 0.15, W3: 0.15, W4: 0.3,
		ScoreOn:      0.5, // ИСПРАВЛЕНИЕ: ослаблены пороги
		ScoreOff:     0.3,
		MinFramesOn:  3, // ИСПРАВЛЕНИЕ: уменьшены для более дробной сегментации
		MaxFramesOff: 2,

		// ИСПРАВЛЕНИЕ: эмпирические масштабные коэффициенты
		SNRScale:         20.0,
		CrestScale:       10.0,
		SpeechRatioScale: 10.0,
	}
}

// внутренний тип сегмента
type seg struct {
	start int
	end   int
	conf  float64
}

// DetectSpeechSegments загружает CSV и возвращает речевые сегменты для всех потоков и каналов.
func DetectSpeechSegments(csvPath string, cfg VADConfig) ([]SpeechSegment, error) {
	lm, err := streammap.LoudnessMapFromCSV(csvPath)
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}

	var allSegments []SpeechSegment

	for topKey, metrics := range lm.Data {
		if !strings.HasSuffix(topKey, "_speech") {
			continue
		}

		streamStr := strings.TrimSuffix(topKey, "_speech")
		streamIdx, err := parseInt(streamStr)
		if err != nil {
			continue
		}

		totalKey := streamStr + "_total"
		totalMetrics, hasTotal := lm.Data[totalKey]
		if !hasTotal {
			totalMetrics = metrics
		}

		channels := extractChannelNumbers(metrics)
		if len(channels) == 0 {
			continue
		}

		for _, ch := range channels {
			rmsSpeech, ok1 := metrics[fmt.Sprintf("%d.RMS_level", ch)]
			maxLevel, ok3 := metrics[fmt.Sprintf("%d.Max_level", ch)]
			if !ok1 || !ok3 {
				continue
			}

			if average(rmsSpeech) < -100.0 {
				continue
			}

			// --- ИСПРАВЛЕНИЕ: адаптивная оценка шума, если Noise_floor отсутствует ---
			noise, ok2 := metrics[fmt.Sprintf("%d.Noise_floor", ch)]
			if !ok2 {
				// оцениваем шум скользящим минимумом RMS за окно 100 кадров
				noise = estimateNoiseFloor(rmsSpeech, 100)
			}

			// --- ИСПРАВЛЕНИЕ: если Max_difference отсутствует, заполняем фиктивными значениями,
			// чтобы переходный признак всегда давал 0 ---
			maxDiff, ok4 := metrics[fmt.Sprintf("%d.Max_difference", ch)]
			if !ok4 {
				// заполняем значением значительно ниже порога, чтобы transient всегда = 0
				maxDiff = make([]float64, len(rmsSpeech))
				for i := range maxDiff {
					maxDiff[i] = -999.0
				}
			}

			// RMS total для этого же канала
			rmsTotal, hasTotalCh := totalMetrics[fmt.Sprintf("%d.RMS_level", ch)]
			if !hasTotalCh {
				rmsTotal = rmsSpeech
			}

			segs := vadSingleChannel(rmsSpeech, noise, maxLevel, maxDiff, rmsTotal, cfg)
			for _, s := range segs {
				allSegments = append(allSegments, SpeechSegment{
					Stream:     streamIdx,
					Channel:    ch,
					StartFrame: s.start,
					EndFrame:   s.end,
					StartSec:   float64(s.start) * streammap.FrameFactor,
					EndSec:     float64(s.end+1) * streammap.FrameFactor,
					Confidence: s.conf,
				})
			}
		}
	}

	sort.Slice(allSegments, func(i, j int) bool {
		if allSegments[i].Stream != allSegments[j].Stream {
			return allSegments[i].Stream < allSegments[j].Stream
		}
		if allSegments[i].StartFrame != allSegments[j].StartFrame {
			return allSegments[i].StartFrame < allSegments[j].StartFrame
		}
		return allSegments[i].Channel < allSegments[j].Channel
	})
	return allSegments, nil
}

// vadSingleChannel выполняет покадровый VAD с нормализованными признаками.
func vadSingleChannel(rmsSpeech, noise, maxLevel, maxDiff, rmsTotal []float64, cfg VADConfig) []seg {
	n := len(rmsSpeech)
	if n == 0 {
		return nil
	}

	const silenceThresholdDB = -60.0 // ниже этого считаем тишиной
	const silenceProbability = 0.01  // уверенность, что это речь

	scores := make([]float64, n)

	for i := 0; i < n; i++ {
		// Явная тишина
		if rmsSpeech[i] < silenceThresholdDB {
			scores[i] = silenceProbability
			continue
		}

		// Подготовка данных (замена -inf)
		rs := rmsSpeech[i]
		if rs < -700 {
			rs = -120
		}
		nf := noise[i]
		if nf < -700 {
			nf = -120
		}
		ml := maxLevel[i]
		if ml < -700 {
			ml = -120
		}
		md := maxDiff[i]
		if md < -700 {
			md = -120
		}
		rt := rmsTotal[i]
		if rt < -700 {
			rt = -120
		}

		// Признаки
		snr := rs - nf
		cr := ml - rs
		speechRatio := rs - rt

		snrNorm := (snr - cfg.SNRThreshold) / cfg.SNRScale
		if snr < 0 {
			snrNorm = snr / cfg.SNRScale
		}
		crNorm := (cr - cfg.CrestTypical) / cfg.CrestScale
		srNorm := (speechRatio - cfg.SpeechRatioThreshold) / cfg.SpeechRatioScale

		transientNorm := 0.0
		if md > cfg.TransientThr {
			transientNorm = 1.0
		}

		z := cfg.W1*snrNorm + cfg.W2*crNorm + cfg.W3*transientNorm + cfg.W4*srNorm
		scores[i] = sigmoid(z)
		if rt < -40.0 {
			scores[i] = math.Min(scores[i], 0.1) // сильно штрафуем
		}
	}

	smoothed := medianFilter(scores, 3)
	return extractSegments(smoothed, cfg.ScoreOn, cfg.ScoreOff, cfg.MinFramesOn, cfg.MaxFramesOff)
}

// --- ИСПРАВЛЕНИЕ: новая функция оценки шума через скользящий минимум ---
func estimateNoiseFloor(rms []float64, window int) []float64 {
	n := len(rms)
	noise := make([]float64, n)
	const silence = -750.0
	for i := 0; i < n; i++ {
		start := max(0, i-window/2)
		end := min(n, i+window/2)
		minVal := math.MaxFloat64
		for j := start; j < end; j++ {
			if rms[j] > silence && rms[j] < minVal {
				minVal = rms[j]
			}
		}
		if minVal == math.MaxFloat64 {
			minVal = -120.0
		}
		noise[i] = minVal
	}
	return noise
}

// --- Остальные вспомогательные функции без изменений (кроме констант) ---

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func medianFilter(data []float64, window int) []float64 {
	n := len(data)
	res := make([]float64, n)
	half := window / 2
	for i := 0; i < n; i++ {
		lo := max(0, i-half)
		hi := min(n, i+half+1)
		slice := make([]float64, hi-lo)
		copy(slice, data[lo:hi])
		sort.Float64s(slice)
		res[i] = slice[len(slice)/2]
	}
	return res
}

func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func extractChannelNumbers(metrics map[string][]float64) []int {
	set := make(map[int]struct{})
	for k := range metrics {
		parts := splitMetricKey(k)
		if len(parts) == 2 {
			ch, err := parseInt(parts[0])
			if err == nil {
				set[ch] = struct{}{}
			}
		}
	}
	chs := make([]int, 0, len(set))
	for ch := range set {
		chs = append(chs, ch)
	}
	sort.Ints(chs)
	return chs
}

func splitMetricKey(key string) []string {
	idx := strings.IndexByte(key, '.')
	if idx == -1 {
		return nil
	}
	return []string{key[:idx], key[idx+1:]}
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// extractSegments выделяет непрерывные участки речи с гистерезисом.
func extractSegments(probs []float64, onThr, offThr float64, minOn, maxOff int) []seg {
	n := len(probs)
	if n == 0 {
		return nil
	}
	state := make([]int, n)
	for i, v := range probs {
		if v > onThr {
			state[i] = 1
		} else if v < offThr {
			state[i] = 0
		} else {
			state[i] = -1
		}
	}
	// гистерезис
	for i := 0; i < n; i++ {
		if state[i] == -1 {
			left, right := -1, -1
			for j := i - 1; j >= 0; j-- {
				if state[j] != -1 {
					left = state[j]
					break
				}
			}
			for j := i + 1; j < n; j++ {
				if state[j] != -1 {
					right = state[j]
					break
				}
			}
			if left == 1 || right == 1 {
				state[i] = 1
			} else {
				state[i] = 0
			}
		}
	}
	// удаление коротких включений
	runStart := -1
	for i := 0; i <= n; i++ {
		if i < n && state[i] == 1 {
			if runStart == -1 {
				runStart = i
			}
		} else {
			if runStart != -1 {
				if i-runStart < minOn {
					for j := runStart; j < i; j++ {
						state[j] = 0
					}
				}
				runStart = -1
			}
		}
	}
	// склеивание коротких пауз
	zeroStart := -1
	for i := 0; i <= n; i++ {
		if i < n && state[i] == 0 {
			if zeroStart == -1 {
				zeroStart = i
			}
		} else {
			if zeroStart != -1 {
				length := i - zeroStart
				if length < maxOff && zeroStart > 0 && i < n && state[zeroStart-1] == 1 && state[i] == 1 {
					for j := zeroStart; j < i; j++ {
						state[j] = 1
					}
				}
				zeroStart = -1
			}
		}
	}
	// сбор сегментов
	var segments []seg
	inSeg := false
	start := 0
	for i := 0; i <= n; i++ {
		if i < n && state[i] == 1 {
			if !inSeg {
				start = i
				inSeg = true
			}
		} else {
			if inSeg {
				end := i - 1
				avg := average(probs[start : end+1])
				segments = append(segments, seg{start, end, avg})
				inSeg = false
			}
		}
	}
	return segments
}
