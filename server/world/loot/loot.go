package loot

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/item/potion"
	"github.com/df-mc/dragonfly/server/world"
)

//go:embed loot_tables/*
var lootFS embed.FS //

// Generate loads a loot table from the embedded filesystem and generates items.
func Generate(path string) ([]item.Stack, bool) {
	// We no longer prefix with "server/world/loot/".
	// The path passed should be relative to the loot_tables folder (e.g., "chests/dungeon.json").
	t, err := LoadTable(path)
	if err != nil {
		fmt.Printf("[Loot System] Error loading table '%s': %v\n", path, err)
		return nil, false
	}
	return t.Generate(), true
}

// LoadTable reads the JSON data directly from the embedded memory.
func LoadTable(path string) (LootTable, error) {
	// b, err := os.ReadFile(path) is replaced by:
	b, err := lootFS.ReadFile(path)
	if err != nil {
		return LootTable{}, err
	}
	var t LootTable
	err = json.Unmarshal(b, &t)
	return t, err
}

// Generate processes the entire LootTable and returns a slice of all stacks generated.
func (t LootTable) Generate() []item.Stack {
	var stacks []item.Stack
	for _, p := range t.Pools {
		rolls := RollValue(p.Rolls)
		for i := 0; i < rolls; i++ {
			if s, ok := p.rollEntry(); ok {
				stacks = append(stacks, s)
			}
		}
	}
	return stacks
}

// --- Struct Definitions ---

type LootTable struct {
	Pools []Pool `json:"pools"`
}

type Pool struct {
	Rolls   Value   `json:"rolls"`
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Weight    int        `json:"weight"`
	Functions []Function `json:"functions,omitempty"`
}

type Function struct {
	Function string          `json:"function"`
	Count    Value           `json:"count"`
	Levels   Value           `json:"levels"`
	ID       string          `json:"id"`
	Enchants []EnchantConfig `json:"enchants"`
}

type EnchantConfig struct {
	ID    string `json:"id"`
	Level Value  `json:"level"`
}

type Value struct {
	Min, Max int
}

func (v *Value) UnmarshalJSON(data []byte) error {
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		v.Min, v.Max = i, i
		return nil
	}
	var m struct {
		Min int `json:"min"`
		Max int `json:"max"`
	}
	if err := json.Unmarshal(data, &m); err == nil {
		v.Min, v.Max = m.Min, m.Max
		return nil
	}
	return nil
}

// --- Logic ---

func (p *Pool) rollEntry() (item.Stack, bool) {
	totalWeight := 0
	for _, e := range p.Entries {
		if e.Weight == 0 {
			e.Weight = 1
		}
		totalWeight += e.Weight
	}
	if totalWeight <= 0 {
		return item.Stack{}, false
	}

	r := rand.Intn(totalWeight)
	current := 0

	for _, e := range p.Entries {
		current += e.Weight
		if r < current {
			if e.Type != "item" {
				return item.Stack{}, false
			}

			name := strings.TrimPrefix(e.Name, "minecraft:")
			it, ok := world.ItemByName("minecraft:"+name, 0)
			if !ok {
				fmt.Printf("[Loot System] Item not found: %s\n", name)
				return item.Stack{}, false
			}

			count := 1
			for _, f := range e.Functions {
				if f.Function == "set_count" {
					count = RollValue(f.Count)
				}
			}

			s := item.NewStack(it, count)

			for _, f := range e.Functions {
				switch f.Function {
				case "enchant_randomly":
					s = applyRandomEnchant(s)
				case "enchant_with_levels":
					s = applyEnchantWithLevels(s, RollValue(f.Levels))
				case "specific_enchants":
					for _, spec := range f.Enchants {
						if enc, ok := enchantmentByName(spec.ID); ok {
							s = s.WithEnchantments(item.NewEnchantment(enc, RollValue(spec.Level)))
						}
					}
				case "set_potion":
					if pot, ok := potionByName(f.ID); ok {
						if _, ok := s.Item().(item.Potion); ok {
							s = item.NewStack(item.Potion{Type: pot}, s.Count())
						} else if _, ok := s.Item().(item.SplashPotion); ok {
							s = item.NewStack(item.SplashPotion{Type: pot}, s.Count())
						}
					}
				}
			}
			return s, true
		}
	}
	return item.Stack{}, false
}

func RollValue(v Value) int {
	if v.Max <= v.Min {
		return v.Min
	}
	return rand.Intn(v.Max-v.Min+1) + v.Min
}

// --- Registries ---

func enchantmentByName(name string) (item.EnchantmentType, bool) {
	name = strings.ToLower(strings.TrimPrefix(name, "minecraft:"))
	m := map[string]item.EnchantmentType{
		"protection":            enchantment.Protection,
		"fire_protection":       enchantment.FireProtection,
		"feather_falling":       enchantment.FeatherFalling,
		"blast_protection":      enchantment.BlastProtection,
		"projectile_protection": enchantment.ProjectileProtection,
		"thorns":                enchantment.Thorns,
		"respiration":           enchantment.Respiration,
		"depth_strider":         enchantment.DepthStrider,
		"aqua_affinity":         enchantment.AquaAffinity,
		"sharpness":             enchantment.Sharpness,
		"knockback":             enchantment.Knockback,
		"fire_aspect":           enchantment.FireAspect,
		"efficiency":            enchantment.Efficiency,
		"silk_touch":            enchantment.SilkTouch,
		"unbreaking":            enchantment.Unbreaking,
		"fortune":               enchantment.Fortune,
		"power":                 enchantment.Power,
		"punch":                 enchantment.Punch,
		"flame":                 enchantment.Flame,
		"infinity":              enchantment.Infinity,
		"mending":               enchantment.Mending,
		"curse_of_vanishing":    enchantment.CurseOfVanishing,
		"multishot":             enchantment.Multishot,
		"quick_charge":          enchantment.QuickCharge,
		"soul_speed":            enchantment.SoulSpeed,
		"swift_sneak":           enchantment.SwiftSneak,
	}
	e, ok := m[name]
	return e, ok
}

func getAllEnchantments() []item.EnchantmentType {
	return []item.EnchantmentType{
		enchantment.Protection, enchantment.FireProtection, enchantment.FeatherFalling,
		enchantment.BlastProtection, enchantment.ProjectileProtection, enchantment.Thorns,
		enchantment.Respiration, enchantment.DepthStrider, enchantment.AquaAffinity,
		enchantment.Sharpness, enchantment.Knockback, enchantment.FireAspect,
		enchantment.Efficiency, enchantment.SilkTouch, enchantment.Unbreaking,
		enchantment.Fortune, enchantment.Power, enchantment.Punch,
		enchantment.Flame, enchantment.Infinity, enchantment.Mending,
		enchantment.CurseOfVanishing, enchantment.Multishot, enchantment.QuickCharge,
		enchantment.SoulSpeed, enchantment.SwiftSneak,
	}
}

func potionByName(name string) (potion.Potion, bool) {
	name = strings.ToLower(strings.TrimPrefix(name, "minecraft:"))
	m := map[string]potion.Potion{
		"water":                potion.Water(),
		"mundane":              potion.Mundane(),
		"long_mundane":         potion.LongMundane(),
		"thick":                potion.Thick(),
		"awkward":              potion.Awkward(),
		"night_vision":         potion.NightVision(),
		"long_night_vision":    potion.LongNightVision(),
		"invisibility":         potion.Invisibility(),
		"long_invisibility":    potion.LongInvisibility(),
		"leaping":              potion.Leaping(),
		"long_leaping":         potion.LongLeaping(),
		"strong_leaping":       potion.StrongLeaping(),
		"fire_resistance":      potion.FireResistance(),
		"long_fire_resistance": potion.LongFireResistance(),
		"swiftness":            potion.Swiftness(),
		"long_swiftness":       potion.LongSwiftness(),
		"strong_swiftness":     potion.StrongSwiftness(),
		"slowness":             potion.Slowness(),
		"long_slowness":        potion.LongSlowness(),
		"strong_slowness":      potion.StrongSlowness(),
		"water_breathing":      potion.WaterBreathing(),
		"long_water_breathing": potion.LongWaterBreathing(),
		"healing":              potion.Healing(),
		"strong_healing":       potion.StrongHealing(),
		"harming":              potion.Harming(),
		"strong_harming":       potion.StrongHarming(),
		"poison":               potion.Poison(),
		"long_poison":          potion.LongPoison(),
		"strong_poison":        potion.StrongPoison(),
		"regeneration":         potion.Regeneration(),
		"long_regeneration":    potion.LongRegeneration(),
		"strong_regeneration":  potion.StrongRegeneration(),
		"strength":             potion.Strength(),
		"long_strength":        potion.LongStrength(),
		"strong_strength":      potion.StrongStrength(),
		"weakness":             potion.Weakness(),
		"long_weakness":        potion.LongWeakness(),
		"wither":               potion.Wither(),
		"turtle_master":        potion.TurtleMaster(),
		"long_turtle_master":   potion.LongTurtleMaster(),
		"strong_turtle_master": potion.StrongTurtleMaster(),
		"slow_falling":         potion.SlowFalling(),
		"long_slow_falling":    potion.LongSlowFalling(),
	}
	p, ok := m[name]
	return p, ok
}

// --- Application Helpers ---

func applyRandomEnchant(s item.Stack) item.Stack {
	var valid []item.EnchantmentType
	for _, enc := range getAllEnchantments() {
		if enc.CompatibleWithItem(s.Item()) {
			valid = append(valid, enc)
		}
	}
	if len(valid) > 0 {
		e := valid[rand.Intn(len(valid))]
		return s.WithEnchantments(item.NewEnchantment(e, 1))
	}
	return s
}

func applyEnchantWithLevels(s item.Stack, levels int) item.Stack {
	for _, enc := range getAllEnchantments() {
		if enc.CompatibleWithItem(s.Item()) {
			if levels > 0 {
				max := enc.MaxLevel()
				lvl := 1
				if levels > 15 && max > 1 {
					lvl = rand.Intn(max) + 1
				}
				return s.WithEnchantments(item.NewEnchantment(enc, lvl))
			}
		}
	}
	return s
}
