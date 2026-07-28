// Package blocksize provides block size calculation for video frames,
// supporting multiple selection policies with tie‑breaking based on aspect ratio.
package blocksize

import (
	"errors"
	"math"
	"sort"
)

// Default values used across configuration helpers.
const (
	defaultMinBlockSize = 8
	defaultMaxBlocks    = 14400
)

// Grid describes a complete tiling of a frame into blocks.
type Grid struct {
	HorizontalBlocks int // number of blocks along the width
	VerticalBlocks   int // number of blocks along the height
	BlockWidth       int // block width in pixels
	BlockHeight      int // block height in pixels
}

// TotalBlocks returns the total number of blocks in the grid.
func (g Grid) TotalBlocks() int { return g.HorizontalBlocks * g.VerticalBlocks }

// Policy defines the strategy for selecting the best block size.
type Policy int

const (
	// PolicyMaxBlocks picks the size that yields the maximum number of blocks
	// (minimum block area) within the MaxBlocksPerFrame limit.
	PolicyMaxBlocks Policy = iota

	// PolicyMinBlocks picks the size that yields the minimum number of blocks
	// (maximum block area) without exceeding MaxBlocksPerFrame.
	PolicyMinBlocks

	// PolicyClosestBlocks picks the size whose block count is closest to TargetBlocks,
	// within the relative BlockTolerance. When distances are equal, it maximizes block count.
	PolicyClosestBlocks

	// PolicyPreserveAspect filters sizes where the absolute difference between the block
	// aspect ratio (bw/bh) and the target aspect ratio is ≤ AspectTolerance, then applies
	// PolicyMaxBlocks on the remaining candidates.
	PolicyPreserveAspect

	// PolicyMaxBlocksPreserveAspect picks the size with the maximum block count.
	// If several sizes share the same maximum count, the one with the aspect ratio
	// closest to TargetAspect (or the frame aspect if TargetAspect == 0) is chosen.
	PolicyMaxBlocksPreserveAspect

	// PolicyMaxBlocksSquare picks the size with the maximum block count.
	// Ties are broken by choosing the size that is closest to a square shape (aspect ratio = 1).
	PolicyMaxBlocksSquare

	// PolicyClosestBlocksPreserveAspect picks the size whose block count is closest
	// to TargetBlocks within the relative BlockTolerance. When multiple sizes are
	// equally close, the one whose aspect ratio is nearest to TargetAspect
	// (or the frame aspect if TargetAspect == 0) wins.
	PolicyClosestBlocksPreserveAspect
)

// CalculationConfig holds all parameters for computing block size.
type CalculationConfig struct {
	// Minimum allowed block dimensions (0 means no constraint).
	MinBlockWidth  int
	MinBlockHeight int

	// Maximum total number of blocks allowed (0 means no limit).
	MaxBlocksPerFrame int

	// Policy for selecting the optimal size.
	Policy Policy

	// Parameters for PolicyClosestBlocks and PolicyClosestBlocksPreserveAspect.
	TargetBlocks   int     // desired number of blocks (must be > 0)
	BlockTolerance float64 // relative tolerance, e.g., 0.1 = ±10%

	// Parameter for PolicyPreserveAspect.
	AspectTolerance float64 // maximum absolute difference between bw/bh and target aspect

	// Target aspect ratio used by PolicyPreserveAspect, PolicyMaxBlocksPreserveAspect,
	// and PolicyClosestBlocksPreserveAspect.
	// If 0, the aspect ratio of the original frame (width/height) is used.
	TargetAspect float64

	// If true, only square blocks (bw == bh) are considered.
	SquareOnly bool

	// Block size alignment: only bw and bh that are multiples of this value are allowed (>0).
	BlockSizeMultiple int
}

// DefaultCfg returns a configuration that balances detail and performance:
// min block 8×8, max 14400 blocks, and among the most blocks it preserves
// the frame aspect ratio as much as possible (PolicyMaxBlocksPreserveAspect).
func DefaultCfg() CalculationConfig {
	return ConfigStandardGrid(defaultMaxBlocks)
}

// Sentinal errors returned by public functions.
var (
	ErrInvalidDimensions = errors.New("width and height must be positive")
	ErrNoValidBlockSize  = errors.New("no block size satisfies all constraints")
)

// dims holds the frame dimensions, used internally to reduce argument count.
type dims struct{ width, height int }

// AllValidGrids returns every possible grid partition of the frame that fully tiles it,
// subject to:
//   - block dimensions are at least 8×8,
//   - the total number of blocks is ≥ 2.
//
// The result is sorted by ascending total block count.
func AllValidGrids(frameWidth, frameHeight int) ([]Grid, error) {
	if frameWidth <= 0 || frameHeight <= 0 {
		return nil, ErrInvalidDimensions
	}
	d := dims{frameWidth, frameHeight}

	candidates := generateAllGrids(d)

	var filtered []Grid
	for _, g := range candidates {
		if g.BlockWidth < 8 || g.BlockHeight < 8 {
			continue
		}
		if g.TotalBlocks() < 2 {
			continue
		}
		filtered = append(filtered, g)
	}

	if len(filtered) == 0 {
		return nil, ErrNoValidBlockSize
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].TotalBlocks() < filtered[j].TotalBlocks()
	})
	return filtered, nil
}

// CalculateBlockSizes computes the optimal block layout according to the configuration.
// It returns a Grid struct with block dimensions and the number of horizontal/vertical blocks.
func CalculateBlockSizes(cfg CalculationConfig, frameWidth, frameHeight int) (Grid, error) {
	d := dims{frameWidth, frameHeight}
	grids, err := AllValidGrids(d.width, d.height)
	if err != nil {
		return Grid{}, err
	}

	return applyPolicy(grids, cfg, d)
}

// generateAllGrids produces every possible Grid that tiles the frame exactly.
func generateAllGrids(d dims) []Grid {
	widthDivs := divisors(d.width)
	heightDivs := divisors(d.height)
	var grids []Grid
	for _, bw := range widthDivs {
		grids = append(grids, gridsForWidth(bw, heightDivs, d)...)
	}
	return grids
}

// gridsForWidth creates grid candidates for a fixed block width bw across all height divisors.
func gridsForWidth(bw int, heightDivs []int, d dims) []Grid {
	var res []Grid
	cols := d.width / bw
	for _, bh := range heightDivs {
		rows := d.height / bh
		res = append(res, Grid{
			BlockWidth:       bw,
			BlockHeight:      bh,
			HorizontalBlocks: cols,
			VerticalBlocks:   rows,
		})
	}
	return res
}

// applyPolicy selects the best Grid from the list according to the configured policy.
func applyPolicy(grids []Grid, cfg CalculationConfig, d dims) (Grid, error) {
	// First, filter by additional constraints from the configuration
	candidates := filterGridsByConstraints(grids, cfg)

	if len(candidates) == 0 {
		return Grid{}, ErrNoValidBlockSize
	}

	switch cfg.Policy {
	case PolicyMaxBlocks:
		return pickMaxBlocks(candidates)
	case PolicyMinBlocks:
		return pickMinBlocks(candidates)
	case PolicyClosestBlocks:
		if cfg.TargetBlocks <= 0 {
			return Grid{}, errors.New("TargetBlocks must be > 0 for PolicyClosestBlocks")
		}
		return pickClosestBlocks(candidates, cfg.TargetBlocks, cfg.BlockTolerance)
	case PolicyPreserveAspect:
		refAspect := effectiveAspect(cfg.TargetAspect, d)
		return pickPreserveAspect(candidates, refAspect, cfg.AspectTolerance)
	case PolicyMaxBlocksPreserveAspect:
		refAspect := effectiveAspect(cfg.TargetAspect, d)
		return pickMaxBlocksPreserveAspect(candidates, refAspect)
	case PolicyMaxBlocksSquare:
		return pickMaxBlocksSquare(candidates)
	case PolicyClosestBlocksPreserveAspect:
		if cfg.TargetBlocks <= 0 {
			return Grid{}, errors.New("TargetBlocks must be > 0 for PolicyClosestBlocksPreserveAspect")
		}
		refAspect := effectiveAspect(cfg.TargetAspect, d)
		return pickClosestBlocksPreserveAspect(candidates, cfg.TargetBlocks, cfg.BlockTolerance, refAspect)
	default:
		return pickMaxBlocks(candidates)
	}
}

// filterGridsByConstraints applies MinBlockWidth/Height, SquareOnly, MaxBlocksPerFrame, BlockSizeMultiple.
func filterGridsByConstraints(grids []Grid, cfg CalculationConfig) []Grid {
	var result []Grid
	for _, g := range grids {
		if cfg.MinBlockWidth > 0 && g.BlockWidth < cfg.MinBlockWidth {
			continue
		}
		if cfg.MinBlockHeight > 0 && g.BlockHeight < cfg.MinBlockHeight {
			continue
		}
		if cfg.SquareOnly && g.BlockWidth != g.BlockHeight {
			continue
		}
		if cfg.BlockSizeMultiple > 0 {
			if g.BlockWidth%cfg.BlockSizeMultiple != 0 || g.BlockHeight%cfg.BlockSizeMultiple != 0 {
				continue
			}
		}
		if cfg.MaxBlocksPerFrame > 0 && g.TotalBlocks() > cfg.MaxBlocksPerFrame {
			continue
		}
		result = append(result, g)
	}
	return result
}

// effectiveAspect returns the aspect ratio to compare against. If target is zero,
// it falls back to the frame aspect ratio.
func effectiveAspect(target float64, d dims) float64 {
	if target == 0 {
		return float64(d.width) / float64(d.height)
	}
	return target
}

// --- Policy pickers (all now operate on Grid) ------------------------------------

// pickMaxBlocks returns the grid with the highest total block count.
// Ties are broken by larger block area.
func pickMaxBlocks(grids []Grid) (Grid, error) {
	best := grids[0]
	bestBlocks := best.TotalBlocks()
	bestArea := best.BlockWidth * best.BlockHeight

	for _, g := range grids[1:] {
		blocks := g.TotalBlocks()
		area := g.BlockWidth * g.BlockHeight
		if blocks > bestBlocks || (blocks == bestBlocks && area > bestArea) {
			best = g
			bestBlocks = blocks
			bestArea = area
		}
	}
	return best, nil
}

// pickMinBlocks returns the grid with the lowest total block count.
// Ties are broken by larger block area.
func pickMinBlocks(grids []Grid) (Grid, error) {
	best := grids[0]
	bestBlocks := best.TotalBlocks()
	bestArea := best.BlockWidth * best.BlockHeight

	for _, g := range grids[1:] {
		blocks := g.TotalBlocks()
		area := g.BlockWidth * g.BlockHeight
		if blocks < bestBlocks || (blocks == bestBlocks && area > bestArea) {
			best = g
			bestBlocks = blocks
			bestArea = area
		}
	}
	return best, nil
}

// pickClosestBlocks selects the grid whose block count is closest to target,
// within the relative tolerance. Ties are broken by higher block count.
func pickClosestBlocks(grids []Grid, target int, tolerance float64) (Grid, error) {
	var bestGrid *Grid
	bestDist := math.MaxFloat64
	bestBlocks := -1

	for _, g := range grids {
		relDist := math.Abs(float64(g.TotalBlocks()-target)) / float64(target)
		if relDist <= tolerance {
			dist := math.Abs(float64(g.TotalBlocks() - target))
			if dist < bestDist || (dist == bestDist && g.TotalBlocks() > bestBlocks) {
				cpy := g
				bestGrid = &cpy
				bestDist = dist
				bestBlocks = g.TotalBlocks()
			}
		}
	}

	if bestGrid == nil {
		return Grid{}, ErrNoValidBlockSize
	}
	return *bestGrid, nil
}

// pickPreserveAspect filters grids whose aspect ratio |bw/bh - refAspect| ≤ tolerance,
// then applies the max‑blocks policy on the remaining set.
func pickPreserveAspect(grids []Grid, refAspect, tolerance float64) (Grid, error) {
	var filtered []Grid
	for _, g := range grids {
		blockAspect := float64(g.BlockWidth) / float64(g.BlockHeight)
		if math.Abs(blockAspect-refAspect) <= tolerance {
			filtered = append(filtered, g)
		}
	}
	if len(filtered) == 0 {
		return Grid{}, ErrNoValidBlockSize
	}
	return pickMaxBlocks(filtered)
}

// pickMaxBlocksPreserveAspect picks the grid with the highest block count.
// Ties are resolved by selecting the one whose aspect ratio is closest to refAspect.
func pickMaxBlocksPreserveAspect(grids []Grid, refAspect float64) (Grid, error) {
	best := grids[0]
	bestBlocks := best.TotalBlocks()
	bestAspectDist := math.Abs(float64(best.BlockWidth)/float64(best.BlockHeight) - refAspect)

	for _, g := range grids[1:] {
		blocks := g.TotalBlocks()
		aspectDist := math.Abs(float64(g.BlockWidth)/float64(g.BlockHeight) - refAspect)
		if blocks > bestBlocks || (blocks == bestBlocks && aspectDist < bestAspectDist) {
			best = g
			bestBlocks = blocks
			bestAspectDist = aspectDist
		}
	}
	return best, nil
}

// pickMaxBlocksSquare picks the grid with the highest block count,
// breaking ties by choosing the most square shape (aspect ratio closest to 1).
func pickMaxBlocksSquare(grids []Grid) (Grid, error) {
	best := grids[0]
	bestBlocks := best.TotalBlocks()
	bestAspectDist := math.Abs(float64(best.BlockWidth)/float64(best.BlockHeight) - 1)

	for _, g := range grids[1:] {
		blocks := g.TotalBlocks()
		aspectDist := math.Abs(float64(g.BlockWidth)/float64(g.BlockHeight) - 1)
		if blocks > bestBlocks || (blocks == bestBlocks && aspectDist < bestAspectDist) {
			best = g
			bestBlocks = blocks
			bestAspectDist = aspectDist
		}
	}
	return best, nil
}

// pickClosestBlocksPreserveAspect picks the grid whose block count is closest to target
// within the relative tolerance. Among equally close counts, the one with aspect ratio
// nearest to refAspect is selected.
func pickClosestBlocksPreserveAspect(grids []Grid, target int, tolerance, refAspect float64) (Grid, error) {
	var bestGrid *Grid
	bestDist := math.MaxFloat64
	bestAspectDist := math.MaxFloat64

	for _, g := range grids {
		relDist := math.Abs(float64(g.TotalBlocks()-target)) / float64(target)
		if relDist <= tolerance {
			dist := math.Abs(float64(g.TotalBlocks() - target))
			aspectDist := math.Abs(float64(g.BlockWidth)/float64(g.BlockHeight) - refAspect)
			if dist < bestDist || (dist == bestDist && aspectDist < bestAspectDist) {
				cpy := g
				bestGrid = &cpy
				bestDist = dist
				bestAspectDist = aspectDist
			}
		}
	}

	if bestGrid == nil {
		return Grid{}, ErrNoValidBlockSize
	}
	return *bestGrid, nil
}

// divisors returns all positive divisors of n in ascending order.
func divisors(n int) []int {
	var divs []int
	for i := 1; i*i <= n; i++ {
		if n%i == 0 {
			divs = append(divs, i)
			if i != n/i {
				divs = append(divs, n/i)
			}
		}
	}
	sort.Ints(divs)
	return divs
}

// --- Configuration helpers --------------------------------------------------------

// ConfigStandardGrid maximizes the number of blocks (min 8×8) while respecting
// maxBlocks (14400 if passed 0 or less), and among the densest partitions it preserves the
// frame aspect ratio as much as possible.
func ConfigStandardGrid(maxBlocks int) CalculationConfig {
	if maxBlocks <= 0 {
		maxBlocks = defaultMaxBlocks
	}
	return CalculationConfig{
		MinBlockWidth:     defaultMinBlockSize,
		MinBlockHeight:    defaultMinBlockSize,
		MaxBlocksPerFrame: maxBlocks,
		Policy:            PolicyMaxBlocksPreserveAspect,
		TargetAspect:      0,
	}
}

// ConfigBalanced maximizes the number of blocks (min 8×8, max 14400) and, among
// the densest partitions, prefers blocks that are as square as possible.
func ConfigBalanced() CalculationConfig {
	return CalculationConfig{
		MinBlockWidth:     defaultMinBlockSize,
		MinBlockHeight:    defaultMinBlockSize,
		MaxBlocksPerFrame: defaultMaxBlocks,
		Policy:            PolicyMaxBlocksSquare,
	}
}

// ConfigSquare returns a configuration that uses the maximum possible number of
// strictly square blocks, but never exceeds maxBlocks.
// Min block size is 8×8. If maxBlocks ≤ 0, no upper limit is applied.
func ConfigSquare(maxBlocks int) CalculationConfig {
	return CalculationConfig{
		MinBlockWidth:     defaultMinBlockSize,
		MinBlockHeight:    defaultMinBlockSize,
		MaxBlocksPerFrame: maxBlocks,
		Policy:            PolicyMaxBlocks,
		SquareOnly:        true,
	}
}

// ConfigExactGridSize tries to split the frame into exactly target number of blocks.
// It preserves the frame aspect ratio among the equally matching counts.
// If no valid grid yields exactly target blocks, CalculateBlockSizes returns an error.
func ConfigExactGridSize(target int) CalculationConfig {
	return CalculationConfig{
		MinBlockWidth:     defaultMinBlockSize,
		MinBlockHeight:    defaultMinBlockSize,
		MaxBlocksPerFrame: 0,
		Policy:            PolicyClosestBlocksPreserveAspect,
		TargetBlocks:      target,
		BlockTolerance:    0,
		TargetAspect:      0,
	}
}

// ConfigMaxDetail returns a configuration for maximum detail: min block 8×8,
// unlimited block count, and among the densest partitions preserves the frame aspect ratio.
func ConfigMaxDetail() CalculationConfig {
	return CalculationConfig{
		MinBlockWidth:     defaultMinBlockSize,
		MinBlockHeight:    defaultMinBlockSize,
		MaxBlocksPerFrame: 0,
		Policy:            PolicyMaxBlocksPreserveAspect,
		TargetAspect:      0,
	}
}

// FrameBufferSize returns the exact size in bytes of the raw Y-plane buffer
// for one frame produced by ffmpeg when outputting rawvideo.
func (g Grid) FrameYBufferSize() int {
	return (g.BlockWidth * g.HorizontalBlocks) * (g.BlockHeight * g.VerticalBlocks)
}
