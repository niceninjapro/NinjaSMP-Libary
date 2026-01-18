package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

// GlowLichen is a luminous block found in caves. It can be harvested with shears
// or silk touch and emits a small amount of light.
type GlowLichen struct {
	replaceable
	transparent
	empty
	sourceWaterDisplacer

	// North, East, South, West, Up, Down represent the faces the lichen is attached to.
	Down, Up, North, South, West, East bool
}

// LightEmissionLevel returns 7, as glow lichen emits a faint light.
func (g GlowLichen) LightEmissionLevel() uint8 {
	return 7
}

// BreakInfo returns the break properties, dropping a Glow Lichen item for every face if silk touched.
func (g GlowLichen) BreakInfo() BreakInfo {
	return newBreakInfo(0.2, alwaysHarvestable, hoeEffective, func(t item.Tool, enchantments []item.Enchantment) []item.Stack {
		for _, ench := range enchantments {
			if ench.Type() == enchantment.SilkTouch {
				// Calculate active faces to determine the drop amount.
				count := 0
				if g.North {
					count++
				}
				if g.East {
					count++
				}
				if g.South {
					count++
				}
				if g.West {
					count++
				}
				if g.Up {
					count++
				}
				if g.Down {
					count++
				}

				return []item.Stack{item.NewStack(GlowLichen{}, count)}
			}
		}
		return nil
	})
}

// EncodeItem ...
func (g GlowLichen) EncodeItem() (name string, meta int16) {
	return "minecraft:glow_lichen", 0
}

// EncodeBlock maps the booleans to the "multi_face_direction_bits" property.
func (g GlowLichen) EncodeBlock() (name string, properties map[string]any) {
	var bits int32
	if g.Down {
		bits |= 1
	}
	if g.Up {
		bits |= 2
	}
	if g.South {
		bits |= 4
	}
	if g.West {
		bits |= 8
	}
	if g.North {
		bits |= 16
	}
	if g.East {
		bits |= 32
	}

	return "minecraft:glow_lichen", map[string]any{"multi_face_direction_bits": bits}
}

func (g GlowLichen) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	clickedBlock := tx.Block(pos)
	targetFace := face.Opposite()

	if existing, ok := clickedBlock.(GlowLichen); ok {
		if !existing.hasFace(targetFace) {
			if g.attemptMerge(pos, existing, targetFace, tx, ctx) {
				return true
			}
		}
		if g.attemptMergeAny(pos, existing, tx, ctx) {
			return true
		}
		pos = pos.Side(face)
	}

	actualPos, actualFace, used := firstReplaceable(tx, pos, face, g)
	if !used {
		return false
	}

	targetBlock := tx.Block(actualPos)
	newTargetFace := actualFace.Opposite()

	if existing, ok := targetBlock.(GlowLichen); ok {
		if g.attemptMerge(actualPos, existing, newTargetFace, tx, ctx) {
			return true
		}
		return g.attemptMergeAny(actualPos, existing, tx, ctx)
	}

	supportPos := actualPos.Side(newTargetFace)
	if !tx.Block(supportPos).Model().FaceSolid(supportPos, actualFace, tx) {
		return false
	}

	newLichen := GlowLichen{}.WithFace(newTargetFace)
	place(tx, actualPos, newLichen, user, ctx)
	return placed(ctx)
}

func (g GlowLichen) attemptMergeAny(pos cube.Pos, existing GlowLichen, tx *world.Tx, ctx *item.UseContext) bool {
	for _, f := range cube.Faces() {
		if !existing.hasFace(f) {
			supportPos := pos.Side(f)
			if tx.Block(supportPos).Model().FaceSolid(supportPos, f.Opposite(), tx) {
				if g.attemptMerge(pos, existing, f, tx, ctx) {
					return true
				}
			}
		}
	}
	return false
}

func (g GlowLichen) hasFace(f cube.Face) bool {
	switch f {
	case cube.FaceUp:
		return g.Up
	case cube.FaceDown:
		return g.Down
	case cube.FaceNorth:
		return g.North
	case cube.FaceSouth:
		return g.South
	case cube.FaceWest:
		return g.West
	case cube.FaceEast:
		return g.East
	}
	return false
}

func (g GlowLichen) attemptMerge(pos cube.Pos, existing GlowLichen, face cube.Face, tx *world.Tx, ctx *item.UseContext) bool {
	newLichen := existing.WithFace(face)
	if newLichen == existing {
		return false
	}
	supportPos := pos.Side(face)
	if !tx.Block(supportPos).Model().FaceSolid(supportPos, face.Opposite(), tx) {
		return false
	}
	tx.SetBlock(pos, newLichen, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.BlockPlace{Block: GlowLichen{}})
	ctx.SubtractFromCount(1)
	return true
}

func (g GlowLichen) WithFace(f cube.Face) GlowLichen {
	switch f {
	case cube.FaceUp:
		g.Up = true
	case cube.FaceDown:
		g.Down = true
	case cube.FaceNorth:
		g.North = true
	case cube.FaceSouth:
		g.South = true
	case cube.FaceWest:
		g.West = true
	case cube.FaceEast:
		g.East = true
	}
	return g
}

func (g GlowLichen) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	changed := false
	checkSupport := func(face cube.Face, attached *bool) {
		if !*attached {
			return
		}
		supportPos := pos.Side(face)
		if !tx.Block(supportPos).Model().FaceSolid(supportPos, face.Opposite(), tx) {
			*attached = false
			changed = true
		}
	}

	checkSupport(cube.FaceDown, &g.Down)
	checkSupport(cube.FaceUp, &g.Up)
	checkSupport(cube.FaceNorth, &g.North)
	checkSupport(cube.FaceSouth, &g.South)
	checkSupport(cube.FaceWest, &g.West)
	checkSupport(cube.FaceEast, &g.East)

	if !changed {
		return
	}

	if !g.Down && !g.Up && !g.North && !g.South && !g.West && !g.East {
		breakBlock(g, pos, tx)
		return
	}
	tx.SetBlock(pos, g, nil)
}

func allGlowLichens() (b []world.Block) {
	for i := 0; i < 64; i++ {
		b = append(b, GlowLichen{
			Down:  i&1 != 0,
			Up:    i&2 != 0,
			South: i&4 != 0,
			West:  i&8 != 0,
			North: i&16 != 0,
			East:  i&32 != 0,
		})
	}
	return
}

func (g GlowLichen) DecodeNBT(data map[string]any) any {
	if v, ok := data["multi_face_direction_bits"]; ok {
		bits := v.(int32)
		g.Down = bits&0x1 != 0
		g.Up = bits&0x2 != 0
		g.North = bits&0x4 != 0
		g.South = bits&0x8 != 0
		g.West = bits&0x10 != 0
		g.East = bits&0x20 != 0
	}
	return g
}

func (g GlowLichen) EncodeNBT() map[string]any {
	var bits int32
	if g.Down {
		bits |= 0x1
	}
	if g.Up {
		bits |= 0x2
	}
	if g.North {
		bits |= 0x4
	}
	if g.South {
		bits |= 0x8
	}
	if g.West {
		bits |= 0x10
	}
	if g.East {
		bits |= 0x20
	}

	return map[string]any{"multi_face_direction_bits": bits}
}
