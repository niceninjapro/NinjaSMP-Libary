package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// SnowLayer is a block that covers the ground. It can be stacked up to 8 layers.
type SnowLayer struct {
	transparent
	replaceable
	// Height is 0-7 (representing 1-8 layers in Bedrock).
	Height int
	// Covered is a required vanilla property for snow layers (covered_bit).
	Covered bool
}

// Model ...
func (s SnowLayer) Model() world.BlockModel {
	return model.SnowLayer{Height: s.Height}
}

// BreakInfo ...
func (s SnowLayer) BreakInfo() BreakInfo {
	return newBreakInfo(0.1, alwaysHarvestable, shovelEffective, func(t item.Tool, e []item.Enchantment) []item.Stack {
		// 1. Handle Silk Touch: Drops itself (1 item per layer), or Snow Block if full.
		if hasSilkTouch(e) {
			if s.Height == 7 {
				// 8 layers (Height 7) drops a Snow Block.
				return []item.Stack{item.NewStack(Snow{}, 1)}
			}
			// Drops 1 item per layer.
			return []item.Stack{item.NewStack(s, s.Height+1)}
		}

		// 2. Handle Shovel Drops (Snowballs).
		if t.ToolType() == item.TypeShovel {
			count := 1
			switch {
			case s.Height >= 7: // 8 layers
				count = 4
			case s.Height >= 5: // 6-7 layers
				count = 3
			case s.Height >= 3: // 4-5 layers
				count = 2
			default: // 1-3 layers (Height 0-2)
				count = 1
			}
			return []item.Stack{item.NewStack(item.Snowball{}, count)}
		}

		// Any other tool drops nothing.
		return nil
	})
}

// UseOnBlock handles placing snow layers. It handles stacking logic and conversion to Snow blocks.
func (s SnowLayer) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	// 1. Check if we are clicking an existing snow layer to stack it.
	// We check the block at the target position first.
	if existing, ok := tx.Block(pos).(SnowLayer); ok {
		if existing.Height < 7 {
			existing.Height++
			place(tx, pos, existing, user, ctx)
			return placed(ctx)
		}
		// If it reaches 8 layers (Height 7) + 1, it becomes a solid Snow block.
		place(tx, pos, Snow{}, user, ctx)
		return placed(ctx)
	}

	// 2. If we aren't clicking ON a snow layer, use standard placement logic
	// to find the position (checking if the clicked face is replaceable).
	pos, _, used := firstReplaceable(tx, pos, face, s)
	if !used {
		return false
	}

	// 3. Check if the block we ended up at is ALREADY a snow layer (e.g. we clicked the side of a block into a layer).
	if existing, ok := tx.Block(pos).(SnowLayer); ok {
		if existing.Height < 7 {
			existing.Height++
			place(tx, pos, existing, user, ctx)
			return placed(ctx)
		}
		place(tx, pos, Snow{}, user, ctx)
		return placed(ctx)
	}

	// 4. Validate support (like Bush).
	if !tx.Block(pos.Side(cube.FaceDown)).Model().FaceSolid(pos.Side(cube.FaceDown), cube.FaceDown.Opposite(), tx) {
		return false
	}

	place(tx, pos, s, user, ctx)
	return placed(ctx)
}

// NeighbourUpdateTick breaks the snow if the block below is removed (Gravity/Support).
func (s SnowLayer) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !tx.Block(pos.Side(cube.FaceDown)).Model().FaceSolid(pos.Side(cube.FaceDown), cube.FaceDown.Opposite(), tx) {
		breakBlock(s, pos, tx)
	}
}

// LiquidRemovable allows water to wash away the snow layer.
func (s SnowLayer) LiquidRemovable() bool {
	return true
}

// HasLiquidDrops ensures that when water breaks this block, it drops items (Snowballs).
func (s SnowLayer) HasLiquidDrops() bool {
	return true
}

// EncodeBlock ...
func (s SnowLayer) EncodeBlock() (string, map[string]any) {
	return "minecraft:snow_layer", map[string]any{
		"height":      int32(s.Height),
		"covered_bit": s.Covered,
	}
}

// DecodeBlock ...
func (s SnowLayer) DecodeBlock(name string, properties map[string]any) (world.Block, bool) {
	if name != "minecraft:snow_layer" {
		return nil, false
	}
	if h, ok := properties["height"]; ok {
		s.Height = int(h.(int32))
	}
	if c, ok := properties["covered_bit"]; ok {
		s.Covered = c.(bool)
	}
	return s, true
}

// EncodeItem ...
func (s SnowLayer) EncodeItem() (name string, meta int16) {
	return "minecraft:snow_layer", 0
}

// allSnowLayers ...
func allSnowLayers() (b []world.Block) {
	for i := 0; i < 8; i++ {
		b = append(b, SnowLayer{Height: i, Covered: false})
		b = append(b, SnowLayer{Height: i, Covered: true})
	}
	return
}
