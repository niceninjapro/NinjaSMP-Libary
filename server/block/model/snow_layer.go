package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// SnowLayer is the model for a snow layer block.
type SnowLayer struct {
	// Height represents the thickness of the snow.
	// 0 = 1 layer (0.125 tall), 7 = 8 layers (1.0 tall/full block).
	Height int
}

// BBox returns the bounding box based on the number of snow layers.
func (s SnowLayer) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	// Calculate the top Y value. Each layer adds 0.125 (1/8th) to the height.
	// Height 0 = 0.125, Height 7 = 1.0.
	maxY := float64(s.Height+1) * 0.125

	// Return a single box spanning the full width (0-1) and the calculated height.
	return []cube.BBox{cube.Box(0, 0, 0, 1, maxY, 1)}
}

// FaceSolid returns whether a face is solid.
func (s SnowLayer) FaceSolid(_ cube.Pos, face cube.Face, _ world.BlockSource) bool {
	// If the snow is a full block (8 layers), all faces are solid.
	if s.Height == 7 {
		return true
	}
	// If it's not a full block, only the bottom face is strictly solid
	// (prevents rendering issues with blocks below it).
	return face == cube.FaceDown
}
