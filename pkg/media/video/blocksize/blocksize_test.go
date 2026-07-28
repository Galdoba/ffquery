package blocksize

import (
	"fmt"
	"testing"
)

func Test_Blocksize(t *testing.T) {
	type size struct{ w, h int }
	for _, exampleSize := range []size{{1920, 1080}} {
		fmt.Println("size", exampleSize)
		configs := []CalculationConfig{
			ConfigStandardGrid(0),
			ConfigBalanced(),
			ConfigExactGridSize(3),
			ConfigMaxDetail(),
			ConfigSquare(150000),
		}
		names := []string{"std", "bal", "exact", "maxDet", "Square"}

		for i, cfg := range configs {
			fmt.Println(names[i])
			grid, err := CalculateBlockSizes(cfg, exampleSize.w, exampleSize.h)
			if err != nil {
				fmt.Println("error:", err)
			} else {
				fmt.Printf("Grid: %+v\n", grid)
			}
		}
	}
	grids, err := AllValidGrids(1920, 1080)
	fmt.Println(err)
	for i, grid := range grids {
		fmt.Println(i, grid)
	}
}
