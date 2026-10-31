package duo

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	druidresto "github.com/wowsims/forever/sim/druid/restoration"
	"github.com/wowsims/forever/sim/hunter"
	"github.com/wowsims/forever/sim/paladin/protection"
	"github.com/wowsims/forever/sim/priest"
	"github.com/wowsims/forever/sim/priest/healer"
	shamanresto "github.com/wowsims/forever/sim/shaman/restoration"
	"github.com/wowsims/forever/sim/warlock"
)

func init() {
	hunter.RegisterHunter()
	priest.RegisterPriest()
	healer.RegisterHealerPriest()
	shamanresto.RegisterRestorationShaman()
	druidresto.RegisterRestorationDruid()
	warlock.RegisterWarlock()
	protection.RegisterProtectionPaladin()
	common.RegisterAllEffects()
}

const ui = "../../ui/specs/"

func loadPlayer(path string) *proto.Player {
	data, _ := os.ReadFile(path)
	var wrap map[string]json.RawMessage
	_ = json.Unmarshal(data, &wrap)
	p := &proto.Player{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrap["player"], p); err != nil { panic(err) }
	return p
}

func addShoot(apl string) string {
	var m map[string]any
	_ = json.Unmarshal([]byte(apl), &m)
	var sh any
	_ = json.Unmarshal([]byte(`{"action":{"castSpell":{"spellId":{"spellId":5019},"target":{"type":"Target"}}}}`), &sh)
	m["priorityList"] = append(m["priorityList"].([]any), sh)
	out, _ := json.Marshal(m)
	return string(out)
}

// ---- heal APL helpers, all aimed at tankJSON ----
func hp(t string) string { return `{"currentHealthPercent":{"sourceUnit":` + t + `}}` }
func le(t, v string) string { return `{"cmp":{"op":"OpLe","lhs":` + hp(t) + `,"rhs":{"const":{"val":"` + v + `"}}}}` }
func mana(v string) string { return `{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"` + v + `"}}}}` }
func heal(t, cond, id string) string {
	return `{"action":{"condition":` + cond + `,"castSpell":{"spellId":{"spellId":` + id + `},"target":` + t + `}}}`
}
func notDot(id, t string) string { return `{"not":{"val":{"dotIsActive":{"spellId":{"spellId":` + id + `},"targetUnit":` + t + `}}}}` }
func and(a, b string) string    { return `{"and":{"vals":[` + a + `,` + b + `]}}` }
func dmg(cond, id string) string {
	return `{"action":{"condition":` + cond + `,"castSpell":{"spellId":{"spellId":` + id + `}}}}`
}
func apl(pre string, pl ...string) string {
	return `{"type":"TypeAPL","prepullActions":[` + pre + `],"priorityList":[` + strings.Join(pl, ",") + `]}`
}

func priestP(t string) *proto.Player {
	p := loadPlayer(ui + "priest/dps/builds/smite_testing.build.json")
	p.Race = proto.Race_RaceTroll
	p.TalentsString = "505025021330011531-00505103"
	p.Consumables = &proto.ConsumesSpec{}
	p.Buffs = &proto.IndividualBuffs{}
	p.DistanceFromTarget = 20
	p.Rotation = core.APLRotationFromJsonString(addShoot(apl(`{"action":{"castSpell":{"spellId":{"spellId":10952}}},"doAtValue":{"const":{"val":"-10s"}}}`,
		heal(t, le(t, "40%"), "10917"), heal(t, le(t, "80%"), "10901"), `{"action":{"condition":`+le(t, "70%")+`,"castSpell":{"spellId":{"spellId":1316991,"tag":1},"target":`+t+`}}}`,
		heal(t, le(t, "65%"), "25314"), heal(t, and(le(t, "90%"), notDot("25315", t)), "25315"),
		dmg(and(mana("50%"), `{"not":{"val":{"dotIsActive":{"spellId":{"spellId":10894}}}}}`), "10894"),
		`{"action":{"condition":`+mana("50%")+`,"castSpell":{"spellId":{"spellId":1316995},"target":{"type":"Target"}}}}`,
		dmg(mana("50%"), "10934"))))
	p.Spec = &proto.Player_HealerPriest{HealerPriest: &proto.HealerPriest{Options: &proto.HealerPriest_Options{ClassOptions: &proto.PriestOptions{Armor: proto.PriestOptions_InnerFire}}}}
	return p
}

func shamanP(t string) *proto.Player {
	p := loadPlayer(ui + "shaman/elemental/builds/testing.build.json")
	p.Race = proto.Race_RaceTroll
	p.TalentsString = "5503331--5530003015013131"
	p.Consumables = &proto.ConsumesSpec{}
	p.HealingModel = nil
	p.Buffs = &proto.IndividualBuffs{}
	p.DistanceFromTarget = 20
	pre := `{"action":{"castSpell":{"spellId":{"spellId":408510}}},"doAtValue":{"const":{"val":"-6s"}}},{"action":{"castSpell":{"spellId":{"spellId":25361}}},"doAtValue":{"const":{"val":"-4.5s"}}},{"action":{"castSpell":{"spellId":{"spellId":10614}}},"doAtValue":{"const":{"val":"-3s"}}},{"action":{"castSpell":{"spellId":{"spellId":10497}}},"doAtValue":{"const":{"val":"-1.5s"}}}`
	p.Rotation = core.APLRotationFromJsonString(apl(pre, heal(t, le(t, "40%"), "10468"), heal(t, le(t, "75%"), "1239243"), heal(t, le(t, "70%"), "25357"),
		dmg(and(mana("50%"), `{"not":{"val":{"dotIsActive":{"spellId":{"spellId":29228}}}}}`), "29228"), dmg(mana("50%"), "15208")))
	p.Spec = &proto.Player_RestorationShaman{RestorationShaman: &proto.RestorationShaman{Options: &proto.RestorationShaman_Options{ClassOptions: &proto.ShamanOptions{}}}}
	return p
}

func druidP(t string) *proto.Player {
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassDruid, Race: proto.Race_RaceTauren,
		Equipment:   core.GetGearSet(ui+"druid/balance/gear_sets", "prebis").GearSet,
		Consumables: &proto.ConsumesSpec{}, Buffs: &proto.IndividualBuffs{},
		TalentsString: "55322201--5050035103113031", DistanceFromTarget: 20, ReactionTimeMs: 100,
		Rotation: core.APLRotationFromJsonString(apl("", heal(t, le(t, "45%"), "18562"), heal(t, le(t, "45%"), "25297"),
			heal(t, and(le(t, "90%"), notDot("25299", t)), "25299"), heal(t, le(t, "80%"), "1238215"),
			heal(t, and(le(t, "70%"), notDot("9858", t)), "9858"), heal(t, le(t, "60%"), "25297"),
			dmg(and(mana("50%"), `{"not":{"val":{"dotIsActive":{"spellId":{"spellId":9835}}}}}`), "9835"), dmg(mana("50%"), "9912"))),
	}, &proto.Player_RestorationDruid{RestorationDruid: &proto.RestorationDruid{Options: &proto.RestorationDruid_Options{ClassOptions: &proto.DruidOptions{}}}})
}

func hunterP() *proto.Player {
	data, _ := os.ReadFile(ui + "hunter/dps/apls/turret.apl.json")
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassHunter, Race: proto.Race_RaceTroll,
		Equipment:   core.GetGearSet(ui+"hunter/dps/gear_sets", "p1").GearSet,
		Consumables: &proto.ConsumesSpec{}, Buffs: &proto.IndividualBuffs{},
		TalentsString: "5320001505101251-0053250001-4", DistanceFromTarget: 25, ReactionTimeMs: 100,
		Rotation: core.APLRotationFromJsonString(string(data)),
	}, &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{PetType: proto.HunterOptions_Turtle, PetUptime: 1}}}})
}

func warlockP() *proto.Player {
	p := loadPlayer(ui + "warlock/dps/builds/sm_ruin.build.json")
	p.Race = proto.Race_RaceUndead
	p.Equipment = core.GetGearSet(ui+"warlock/dps/gear_sets", "preraid").GearSet
	p.Consumables = &proto.ConsumesSpec{}
	p.Buffs = &proto.IndividualBuffs{}
	p.GetWarlock().Options.ClassOptions.Summon = proto.WarlockOptions_Voidwalker
	return p
}

func paladinP() *proto.Player {
	data, _ := os.ReadFile(ui + "paladin/protection/apls/default.apl.json")
	js := strings.Replace(string(data), `"prepullActions": [`, `"prepullActions": [{"action": {"castSpell": {"spellId": {"spellId": 25780}}}, "doAtValue": {"const": {"val": "-20s"}}}, `, 1)
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassPaladin, Race: proto.Race_RaceUndead,
		Equipment:   core.GetGearSet(ui+"paladin/protection/gear_sets", "default").GearSet,
		Consumables: &proto.ConsumesSpec{}, Buffs: &proto.IndividualBuffs{},
		TalentsString: "5-5500013301301551-0522103", Rotation: core.APLRotationFromJsonString(js),
		InFrontOfTarget: true, DistanceFromTarget: 5, ReactionTimeMs: 100,
	}, &proto.Player_ProtectionPaladin{ProtectionPaladin: &proto.ProtectionPaladin{Options: &proto.ProtectionPaladin_Options{ClassOptions: &proto.PaladinOptions{}}}})
}

// ---- duo definitions: player 0 is the "main" (owns the pet / is the tank), player 1 the partner ----
func petRef(owner int) string { return petRefI(owner, 0) }
func petRefI(owner, idx int) string { return fmt.Sprintf(`{"type":"Pet","index":%d,"owner":{"type":"Player","index":%d}}`, idx, owner) }
func plRef(i int) string      { return fmt.Sprintf(`{"type":"Player","index":%d}`, i) }

type duo struct {
	name    string
	players func() []*proto.Player
	tank    *proto.UnitReference
	tankPet bool
}

func petTank(owner int32) *proto.UnitReference { return petTankI(owner, 0) }
func petTankI(owner, idx int32) *proto.UnitReference {
	return &proto.UnitReference{Type: proto.UnitReference_Pet, Index: idx, Owner: &proto.UnitReference{Type: proto.UnitReference_Player, Index: owner}}
}

var duos = []duo{
	{"Hunter + Priest", func() []*proto.Player { return []*proto.Player{hunterP(), priestP(petRef(0))} }, petTank(0), true},
	{"Warlock + Priest", func() []*proto.Player { return []*proto.Player{warlockP(), priestP(petRefI(0, 1))} }, petTankI(0, 1), true},
	{"Hunter + Shaman", func() []*proto.Player { return []*proto.Player{hunterP(), shamanP(petRef(0))} }, petTank(0), true},
	{"Paladin + Priest", func() []*proto.Player { return []*proto.Player{paladinP(), priestP(plRef(0))} }, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0}, false},
	{"Shaman + Warlock", func() []*proto.Player { return []*proto.Player{warlockP(), shamanP(petRefI(0, 1))} }, petTankI(0, 1), true},
	{"Hunter + Warlock", func() []*proto.Player { return []*proto.Player{hunterP(), warlockP()} }, petTank(0), true},
	{"Druid + Hunter", func() []*proto.Player { return []*proto.Player{hunterP(), druidP(petRef(0))} }, petTank(0), true},
}

func encounter(sc string) *proto.Encounter {
	mk := func(n int, lvl int32, dmg float64, hpv float64) *proto.Encounter {
		e := &proto.Encounter{Duration: 300, UseHealth: true}
		for i := 0; i < n; i++ {
			t := core.FreshDefaultTargetConfig()
			t.Level = lvl; t.MinBaseDamage = dmg; t.CanCrush = false; t.ParryHaste = false
			t.Stats = make([]float64, 40); t.Stats[proto.Stat_StatHealth] = hpv
			e.Targets = append(e.Targets, t)
		}
		return e
	}
	switch sc {
	case "1 mob":
		return mk(1, 60, 350, 4365)
	case "3-mob pull":
		e := mk(3, 60, 350, 4365)
		e.UseHealth = false; e.Duration = 40
		return e
	}
	return mk(1, 62, 750, 13000) // elite
}

func TestDuos(t *testing.T) {
	for _, sc := range []string{"1 mob", "3-mob pull", "elite"} {
		for _, d := range duos {
			ps := d.players()
			r := &proto.Raid{Parties: []*proto.Party{{Players: ps, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}, Tanks: []*proto.UnitReference{d.tank}}
			cs := core.ComputeStats(&proto.ComputeStatsRequest{Raid: r, Encounter: encounter(sc)})
			if cs.ErrorResult != "" { t.Fatalf("%s: %s", d.name, cs.ErrorResult) }
			if os.Getenv("DBG") != "" {
				for pi := range ps {
					rs := cs.RaidStats.Parties[0].Players[pi].RotationStats
					for i, a := range append(rs.PrepullActions, rs.PriorityList...) { for _, v := range a.Validations { fmt.Printf("VAL %s p%d %d %s\n", d.name, pi, i, v.Validation) } }
				}
				continue
			}
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: encounter(sc), SimOptions: &proto.SimOptions{Iterations: 1500, RandomSeed: 31}})
			if res.Error != nil { t.Fatalf("%s: %v", d.name, res.Error.Message) }
			pm := res.RaidMetrics.Parties[0].Players
			death := pm[0].ChanceOfDeath
			if d.tankPet && len(pm[0].Pets) > int(d.tank.Index) { death = pm[0].Pets[d.tank.Index].ChanceOfDeath }
			out := fmt.Sprintf("DUO|%s|%s|duoDPS=%.1f|tankDeath=%.3f", sc, d.name, res.RaidMetrics.Dps.Avg, death)
			if sc == "1 mob" { out += fmt.Sprintf("|TTK=%.1fs", 4365/res.RaidMetrics.Dps.Avg) }
			if sc == "elite" { out += fmt.Sprintf("|TTK=%.1fs", 13000/res.RaidMetrics.Dps.Avg) }
			for pi, p := range pm {
				maxMana := cs.RaidStats.Parties[0].Players[pi].FinalStats.Stats[proto.Stat_StatMana]
				var net float64
				for _, rm := range p.Resources { if rm.Type == proto.ResourceType_ResourceTypeMana { net += rm.Gain } }
				net /= 1500
				pct := 0.0
				if maxMana > 0 { pct = -net / maxMana * 100 }
				out += fmt.Sprintf("|p%d manaUsed=%.0f%%|p%d DTPS=%.0f|p%d death=%.3f", pi, pct, pi, p.Dtps.Avg, pi, p.ChanceOfDeath)
			}
			fmt.Println(out)
		}
	}
}
