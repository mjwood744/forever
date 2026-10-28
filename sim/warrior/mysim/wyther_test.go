package mysim

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/shaman/restoration"
	"github.com/wowsims/forever/sim/warlock"
)

func init() { restoration.RegisterRestorationShaman(); warlock.RegisterWarlock() }

func loadPlayer(path string) *proto.Player {
	data, _ := os.ReadFile(path)
	var wrap map[string]json.RawMessage
	_ = json.Unmarshal(data, &wrap)
	p := &proto.Player{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrap["player"], p); err != nil { panic(err) }
	return p
}

const tankHP = `{"currentHealthPercent":{"sourceUnit":{"type":"Player","index":1}}}`
const tankRef = `{"type":"Player","index":1}`

func shamanAPL(gate string, totems bool) string {
	pre := `{"action":{"castSpell":{"spellId":{"spellId":408510}}},"doAtValue":{"const":{"val":"-6s"}}}`
	if totems {
		pre += `,{"action":{"castSpell":{"spellId":{"spellId":25361}}},"doAtValue":{"const":{"val":"-4.5s"}}},{"action":{"castSpell":{"spellId":{"spellId":10614}}},"doAtValue":{"const":{"val":"-3s"}}},{"action":{"castSpell":{"spellId":{"spellId":10497}}},"doAtValue":{"const":{"val":"-1.5s"}}}`
	}
	le := func(v string) string { return `{"cmp":{"op":"OpLe","lhs":` + tankHP + `,"rhs":{"const":{"val":"` + v + `"}}}}` }
	mana := `{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"` + gate + `"}}}}`
	heal := func(cond, id string) string {
		return `{"action":{"condition":` + cond + `,"castSpell":{"spellId":{"spellId":` + id + `},"target":` + tankRef + `}}}`
	}
	pl := []string{
		heal(le("40%"), "10468"),
		heal(le("75%"), "1239243"),
		heal(le("60%"), "10623"),
		heal(le("70%"), "25357"),
		`{"action":{"condition":{"and":{"vals":[` + mana + `,{"not":{"val":{"dotIsActive":{"spellId":{"spellId":29228}}}}}]}},"castSpell":{"spellId":{"spellId":29228}}}}`,
		`{"action":{"condition":` + mana + `,"castSpell":{"spellId":{"spellId":15208}}}}`,
	}
	return `{"type":"TypeAPL","prepullActions":[` + pre + `],"priorityList":[` + strings.Join(pl, ",") + `]}`
}

func wyther(talents, gate string, totems bool) *proto.Player {
	p := loadPlayer("../../../ui/specs/shaman/elemental/builds/testing.build.json")
	p.Race = proto.Race_RaceTroll
	p.TalentsString = talents
	p.Consumables = &proto.ConsumesSpec{}
	p.HealingModel = nil
	p.Buffs = &proto.IndividualBuffs{}
	p.DistanceFromTarget = 20
	p.Rotation = core.APLRotationFromJsonString(shamanAPL(gate, totems))
	p.Spec = &proto.Player_RestorationShaman{RestorationShaman: &proto.RestorationShaman{Options: &proto.RestorationShaman_Options{ClassOptions: &proto.ShamanOptions{}}}}
	return p
}

func malyce() *proto.Player {
	p := loadPlayer("../../../ui/specs/warlock/dps/builds/sm_ruin.build.json")
	p.Race = proto.Race_RaceTroll
	p.Equipment = core.GetGearSet("../../../ui/specs/warlock/dps/gear_sets", "preraid").GearSet
	p.Consumables = &proto.ConsumesSpec{}
	p.Buffs = &proto.IndividualBuffs{}
	return p
}

func trio2(w *proto.Player, skarrTal string, withLock bool) *proto.Raid {
	sk := warrior(wb{"B", skarrTal, true})
	players := []*proto.Player{w, sk}
	if withLock { players = append(players, malyce()) }
	return &proto.Raid{Parties: []*proto.Party{{Players: players, Buffs: &proto.PartyBuffs{BloodPact: true}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
		Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
}

func scen2(sc string) *proto.Encounter {
	switch sc {
	case "1 mob 60s":
		return enc(1, true, 60)
	case "4 mobs 45s":
		return enc(4, true, 45)
	case "trio boss 32k":
		e := enc(1, true, 300)
		e.UseHealth = true
		e.Targets[0].Level = 62; e.Targets[0].MinBaseDamage = 900
		e.Targets[0].Stats = make([]float64, 40); e.Targets[0].Stats[proto.Stat_StatHealth] = 32000
		return e
	}
	e := enc(1, true, 90)
	e.Targets[0].Level = 62; e.Targets[0].MinBaseDamage = 900
	return e
}

func TestWyther(t *testing.T) {
	builds := []struct{ n, tal string }{
		{"A Resto 31 / Ele 20", "5503331--5530003015013131"},
		{"B Resto 31 / Enh 20", "-00523001135-5530003015013131"},
		{"C Ele 31 / Resto 20", "5505331003001311--5530003013"},
	}
	if os.Getenv("DBG") != "" {
		r := trio2(wyther(builds[0].tal, "50%", true), "35300003-05-050531120300021351", true)
		cs := core.ComputeStats(&proto.ComputeStatsRequest{Raid: r, Encounter: scen2("1 mob 60s")})
		if cs.ErrorResult != "" { t.Fatal(cs.ErrorResult) }
		for _, pl := range []int{0, 2} {
			rs := cs.RaidStats.Parties[0].Players[pl].RotationStats
			for i, a := range append(rs.PrepullActions, rs.PriorityList...) { for _, v := range a.Validations { fmt.Printf("VAL p%d %d %s\n", pl, i, v.Validation) } }
		}
		return
	}
	for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k", "stress boss 90s"} {
		for _, b := range builds {
			r := trio2(wyther(b.tal, "50%", true), "35300003-05-050531120300021351", sc != "stress boss 90s")
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: scen2(sc), SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 13}})
			if res.Error != nil { t.Fatalf("%s: %v", b.n, res.Error.Message) }
			ps := res.RaidMetrics.Parties[0].Players
			ld := 0.0
			if len(ps) > 2 { ld = ps[2].Dps.Avg }
			fmt.Printf("WY|%s|%s|wDPS=%.1f|wHPS=%.1f|wOOM=%.1f|skarrDTPS=%.1f|death=%.3f|skarrTPS=%.1f|malyceDPS=%.1f\n", sc, b.n, ps[0].Dps.Avg, ps[0].Hps.Avg, ps[0].SecondsOomAvg, ps[1].Dtps.Avg, ps[1].ChanceOfDeath, ps[1].Threat.Avg, ld)
		}
	}
}
