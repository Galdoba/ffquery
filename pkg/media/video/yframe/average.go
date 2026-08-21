package yframe

import (
	"runtime"
	"sync"

	"github.com/Galdoba/ffquery/pkg/media/video/blocksize"
)


// averageBlock returns the mean Y‑value of the pixels inside a single rectangular block.
// startX and startY are the top‑left coordinates of the block.
func averageBlock(frame []byte, width, blockWidth, blockHeight, startX, startY int) byte {
	var sum int
	for y := 0; y < blockHeight; y++ {
		offset := (startY+y)*width + startX
		for x := 0; x < blockWidth; x++ {
			sum += int(frame[offset+x])
		}
	}
	return byte(sum / (blockWidth * blockHeight))
}

// computeBlockAveragesParallel calculates block averages using multiple goroutines.
// It divides the grid into horizontal strips and processes each strip concurrently.
func computeBlockAveragesParallel(frame []byte, width int, grid blocksize.Grid) []byte {
	numBlocks := grid.HorizontalBlocks * grid.VerticalBlocks
	averages := make([]byte, numBlocks)

	// Determine the number of strips (e.g., number of CPU cores).
	workers := runtime.NumCPU()
	rowsPerWorker := (grid.VerticalBlocks + workers - 1) / workers

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		startRow := w * rowsPerWorker
		if startRow >= grid.VerticalBlocks {
			break
		}
		endRow := startRow + rowsPerWorker
		if endRow > grid.VerticalBlocks {
			endRow = grid.VerticalBlocks
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for row := start; row < end; row++ {
				for col := 0; col < grid.HorizontalBlocks; col++ {
					startY := row * grid.BlockHeight
					startX := col * grid.BlockWidth
					avg := averageBlock(frame, width, grid.BlockWidth, grid.BlockHeight, startX, startY)
					idx := row*grid.HorizontalBlocks + col
					averages[idx] = avg
				}
			}
		}(startRow, endRow)
	}
	wg.Wait()
	return averages
}
