package activities

import (
	"fmt"
	"strings"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// A healer's casts, by spell name, top first. Each heals its target when its health drops below
// `below` percent; a heal over time (`hot`) goes on whenever its aura is off the target.
type healCast struct {
	name  string
	below int
	hot   bool
}

// What each healer spec casts on the tank, top first. The other players get `others` when they drop
// below half health.
type healerPlan struct {
	tank   []healCast
	others string
}

var healerPlans = map[proto.Spec]healerPlan{
	proto.Spec_SpecHealerPriest: {
		tank:   []healCast{{name: "Flash Heal", below: 40}, {name: "Renew", hot: true}, {name: "Greater Heal", below: 60}, {name: "Heal", below: 70}, {name: "Lesser Heal", below: 70}},
		others: "Flash Heal",
	},
	proto.Spec_SpecHolyPaladin: {
		tank:   []healCast{{name: "Holy Light", below: 50}, {name: "Flash of Light", below: 85}},
		others: "Flash of Light",
	},
	proto.Spec_SpecRestorationDruid: {
		tank:   []healCast{{name: "Regrowth", below: 45}, {name: "Rejuvenation", hot: true}, {name: "Healing Touch", below: 70}},
		others: "Rejuvenation",
	},
	proto.Spec_SpecRestorationShaman: {
		tank:   []healCast{{name: "Lesser Healing Wave", below: 40}, {name: "Healing Wave", below: 70}},
		others: "Lesser Healing Wave",
	},
}

// The highest rank of each spell name the player knows, by the spells registered for it.
func knownRanks(have map[int32]bool) map[string]int32 {
	best := map[string]*spelldata.Spell{}
	for id := range have {
		s := spelldata.Find(id)
		if s == nil || s.Name == "" {
			continue
		}
		if b := best[s.Name]; b == nil || s.SpellLevel > b.SpellLevel || (s.SpellLevel == b.SpellLevel && s.ID > b.ID) {
			best[s.Name] = s
		}
	}
	ids := make(map[string]int32, len(best))
	for name, s := range best {
		ids[name] = s.ID
	}
	return ids
}

func unitJSON(self, target int) string {
	if self == target {
		return `{"type":"Self"}`
	}
	return fmt.Sprintf(`{"type":"Player","index":%d}`, target)
}

func healthBelow(unit string, percent int) string {
	return fmt.Sprintf(`{"cmp":{"op":"OpLt","lhs":{"currentHealthPercent":{"sourceUnit":%s}},"rhs":{"const":{"val":"%d%%"}}}}`, unit, percent)
}

func castOn(id int32, unit, cond string) string {
	return fmt.Sprintf(`{"action":{"condition":%s,"castSpell":{"spellId":{"spellId":%d},"target":%s}}}`, cond, id, unit)
}

// A healing rotation for a healer that brings none: the healer specs' pages are gear planners and
// save no rotation. Empty when the player is no healer, already has a rotation, or knows none of its
// spec's heals.
func healerRotation(p *proto.Player, self, tank, groupSize int, have map[int32]bool) *proto.APLRotation {
	if len(p.GetRotation().GetPriorityList()) > 0 {
		return nil
	}
	spec, err := core.PlayerProtoToSpecSafe(p)
	if err != nil {
		return nil
	}
	plan, ok := healerPlans[spec]
	if !ok {
		return nil
	}
	known := knownRanks(have)
	var items []string
	tankUnit := unitJSON(self, tank)
	for _, c := range plan.tank {
		id := known[c.name]
		if id == 0 {
			continue
		}
		if c.hot {
			items = append(items, castOn(id, tankUnit, fmt.Sprintf(`{"not":{"val":{"auraIsActive":{"sourceUnit":%s,"auraId":{"spellId":%d}}}}}`, tankUnit, id)))
		} else {
			items = append(items, castOn(id, tankUnit, healthBelow(tankUnit, c.below)))
		}
	}
	if id := known[plan.others]; id != 0 {
		for i := range groupSize {
			if i != tank {
				items = append(items, castOn(id, unitJSON(self, i), healthBelow(unitJSON(self, i), 50)))
			}
		}
	}
	if len(items) == 0 {
		return nil
	}
	return core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[` + strings.Join(items, ",") + `]}`)
}
