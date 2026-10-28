package mysim

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/priest"
	"github.com/wowsims/forever/sim/priest/healer"
)

func init() { priest.RegisterPriest(); healer.RegisterHealerPriest() }

const healAPL = `{"type":"TypeAPL","prepullActions":[{"action":{"castSpell":{"spellId":{"spellId":10952}}},"doAtValue":{"const":{"val":"-10s"}}}],
"priorityList":[
 {"action":{"condition":{"cmp":{"op":"OpLe","lhs":{"currentHealthPercent":{"sourceUnit":{"type":"Player","index":1}}},"rhs":{"const":{"val":"40%"}}}},"castSpell":{"spellId":{"spellId":10917},"target":{"type":"Player","index":1}}}},
 {"action":{"condition":{"cmp":{"op":"OpLe","lhs":{"currentHealthPercent":{"sourceUnit":{"type":"Player","index":1}}},"rhs":{"const":{"val":"80%"}}}},"castSpell":{"spellId":{"spellId":10901},"target":{"type":"Player","index":1}}}},
 {"action":{"condition":{"cmp":{"op":"OpLe","lhs":{"currentHealthPercent":{"sourceUnit":{"type":"Player","index":1}}},"rhs":{"const":{"val":"70%"}}}},"castSpell":{"spellId":{"spellId":PENHEAL,"tag":1},"target":{"type":"Player","index":1}}}},
 {"action":{"condition":{"cmp":{"op":"OpLe","lhs":{"currentHealthPercent":{"sourceUnit":{"type":"Player","index":1}}},"rhs":{"const":{"val":"65%"}}}},"castSpell":{"spellId":{"spellId":25314},"target":{"type":"Player","index":1}}}},
 {"action":{"condition":{"cmp":{"op":"OpLe","lhs":{"currentHealthPercent":{"sourceUnit":{"type":"Player","index":1}}},"rhs":{"const":{"val":"90%"}}}},"castSpell":{"spellId":{"spellId":25315},"target":{"type":"Player","index":1}}}},
 {"action":{"condition":{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"MANAGATE"}}}},"castSpell":{"spellId":{"spellId":10060}}}},
 {"action":{"condition":{"and":{"vals":[{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"MANAGATE"}}}},{"not":{"val":{"dotIsActive":{"spellId":{"spellId":10894}}}}}]}},"castSpell":{"spellId":{"spellId":10894}}}},
 {"action":{"condition":{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"MANAGATE"}}}},"castSpell":{"spellId":{"spellId":1316995},"target":{"type":"Target"}}}},
 {"action":{"condition":{"and":{"vals":[{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"MANAGATE"}}}},{"not":{"val":{"dotIsActive":{"spellId":{"spellId":15261}}}}}]}},"castSpell":{"spellId":{"spellId":15261}}}},
 {"action":{"condition":{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"MANAGATE"}}}},"castSpell":{"spellId":{"spellId":10934}}}}
]}`

func priestPlayer(talents, penHeal string) *proto.Player {
	data, _ := os.ReadFile("../../../ui/specs/priest/dps/builds/smite_testing.build.json")
	var wrap map[string]json.RawMessage
	_ = json.Unmarshal(data, &wrap)
	p := &proto.Player{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrap["player"], p); err != nil {
		panic(err)
	}
	p.TalentsString = talents
	p.Consumables = &proto.ConsumesSpec{}
	p.Buffs = &proto.IndividualBuffs{GreaterBlessingOfKings: true}
	gate := os.Getenv("GATE"); if gate == "" { gate = "50%" }
	p.Rotation = core.APLRotationFromJsonString(replace(replace(healAPL, "PENHEAL", penHeal), "MANAGATE", gate))
	p.DistanceFromTarget = 20
	p.Spec = &proto.Player_HealerPriest{HealerPriest: &proto.HealerPriest{Options: &proto.HealerPriest_Options{ClassOptions: &proto.PriestOptions{Armor: proto.PriestOptions_InnerFire}}}}
	return p
}

func replace(s, a, b string) string {
	out := ""
	for {
		i := indexOf(s, a)
		if i < 0 {
			return out + s
		}
		out += s[:i] + b
		s = s[i+len(a):]
	}
}
func indexOf(s, a string) int {
	for i := 0; i+len(a) <= len(s); i++ {
		if s[i:i+len(a)] == a {
			return i
		}
	}
	return -1
}

func TestPriestBuilds(t *testing.T) {
	pen := os.Getenv("PENHEAL")
	if pen == "" {
		pen = "1316993"
	}
	builds := []struct{ name, talents string }{
		{"A Smite Disc", "525235031030001001-30515100002-3"},
		{"B Healing Disc", "005025021330011531-335001030301"},
		{"D1 B+PiL (Holy heal side)", "505025021330011531-235001030"},
		{"D2 B+PiL+Divine Fury", "505025021330011531-00505103"},
	}
	tank := player(build{"B", "5-5500013301301551-0522103", true})
	for _, sc := range []struct{ n int; mob bool; lab string; dur float64; lvl, dmg int32 }{{1, true, "1 mob 60s", 60, 60, 350}, {4, true, "4 mobs 45s", 45, 60, 350}, {3, true, "3 mobs 45s", 45, 60, 350}, {1, true, "dungeon boss 90s", 90, 62, 900}} {
		for _, b := range builds {
			party := &proto.Party{Players: []*proto.Player{priestPlayer(b.talents, pen), tank}, Buffs: &proto.PartyBuffs{DevotionAura: true}}
			r := &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
				Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
			if os.Getenv("DBG") != "" {
				cs := core.ComputeStats(&proto.ComputeStatsRequest{Raid: r, Encounter: enc(sc.n, sc.mob)})
				rs := cs.RaidStats.Parties[0].Players[0].RotationStats
				for i, a := range rs.PriorityList { for _, v := range a.Validations { fmt.Printf("VAL %d %s\n", i, v.Validation) } }
				for _, sp := range cs.RaidStats.Parties[0].Players[0].Metadata.Spells { if id := sp.Id.GetSpellId(); id > 1316000 && id < 1317000 { fmt.Printf("SPELL %v\n", sp.Id) } }
				os.Exit(0)
			}
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: encD(sc.n, sc.dur, sc.lvl, sc.dmg), SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 11}})
			if res.Error != nil {
				t.Fatalf("%s: %v", b.name, res.Error.Message)
			}
			pr := res.RaidMetrics.Parties[0].Players[0]
			pa := res.RaidMetrics.Parties[0].Players[1]
			if os.Getenv("DBG") != "" { for _, a := range pr.Actions { c := int32(0); for _, tg := range a.Targets { c += tg.Casts }; fmt.Printf("ACT %v casts=%d\n", a.Id, c) }; for _, w := range res.Logs { _ = w } }
			fmt.Printf("PR|%s|%s|priestDPS=%.1f|HPS=%.1f|oom=%.1f|tankDTPS=%.1f|tankDeath=%.3f|trioDPS=%.1f\n", sc.lab, b.name, pr.Dps.Avg, pr.Hps.Avg, pr.SecondsOomAvg, pa.Dtps.Avg, pa.ChanceOfDeath, pr.Dps.Avg+pa.Dps.Avg)
		}
	}
}

func encD(n int, dur float64, lvl, dmg int32) *proto.Encounter {
	e := enc(n, true)
	e.Duration = dur
	e.DurationVariation = 5
	for _, t := range e.Targets { t.Level = lvl; t.MinBaseDamage = float64(dmg) }
	return e
}
