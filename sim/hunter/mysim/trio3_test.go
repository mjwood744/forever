package mysim

import (
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/druid/restoration"
	"github.com/wowsims/forever/sim/rogue"
)

func init() {
	rogue.RegisterRogue()
	restoration.RegisterRestorationDruid()
}

// Trio #3 Spiteborne: Hawke (Hunter, player 0, the pet tanks), Wyldd (Resto Druid, player 1) healing
// the pet, Shryke (Rogue, player 2).

var rogueBuilds = []struct {
	name, tal, gear string
	bs              bool
}{
	{"Asn Muti", "00532310151501031-302303202014", "p1_muti", false},
	{"Combat Muti", "00532310101401-302303202015152-001", "p1_muti", false},
	{"Combat SS", "005323101012-32030320201515231", "p1_slowmh", false},
	{"Combat BS", "00532310101-32030320201515231-002", "p1_bscombat", true},
	{"Sub Hemo", "005323101014--4323200300013211501", "p1_slowmh", false},
	{"Sub BS", "0053031-3203-4320203300013211051", "p1_bssub", true},
}

func shryke(tal, gear string, backstab bool) *proto.Player {
	rot := core.GetAplRotation("../../../ui/specs/rogue/dps/apls", "default").Rotation
	if backstab {
		for _, v := range rot.ValueVariables {
			if strings.Contains(strings.ToLower(v.Name), "backstab") {
				v.Value = &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: "true"}}}
			}
		}
	}
	return &proto.Player{
		Class: proto.Class_ClassRogue, Race: proto.Race_RaceWindshaperSkyborne,
		Equipment:   core.GetGearSet("../../../ui/specs/rogue/dps/gear_sets", gear).GearSet,
		Consumables: &proto.ConsumesSpec{MhWeaponImbueId: 8928, OhWeaponImbueId: 8985},
		Buffs:       &proto.IndividualBuffs{}, TalentsString: tal, ReactionTimeMs: 100,
		Rotation: rot,
		Spec:     &proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.Rogue_Options{ClassOptions: &proto.RogueOptions{}}}},
	}
}

// wylddAPL heals the unit at tankJSON; Wrath above the mana gate.
func wylddAPL(tankJSON, gate string) string {
	hp := `{"currentHealthPercent":{"sourceUnit":` + tankJSON + `}}`
	le := func(v string) string {
		return `{"cmp":{"op":"OpLe","lhs":` + hp + `,"rhs":{"const":{"val":"` + v + `"}}}}`
	}
	notHot := func(id string) string {
		return `{"not":{"val":{"dotIsActive":{"spellId":{"spellId":` + id + `},"targetUnit":` + tankJSON + `}}}}`
	}
	and := func(a, b string) string { return `{"and":{"vals":[` + a + `,` + b + `]}}` }
	heal := func(cond, id string) string {
		return `{"action":{"condition":` + cond + `,"castSpell":{"spellId":{"spellId":` + id + `},"target":` + tankJSON + `}}}`
	}
	mana := `{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"` + gate + `"}}}}`
	pl := []string{
		heal(le("45%"), "18562"),
		heal(le("45%"), "25297"),
		heal(and(le("90%"), notHot("25299")), "25299"),
		heal(le("80%"), "1238215"),
		heal(and(le("70%"), notHot("9858")), "9858"),
		heal(le("60%"), "25297"),
		`{"action":{"condition":{"and":{"vals":[` + mana + `,{"not":{"val":{"dotIsActive":{"spellId":{"spellId":9835}}}}}]}},"castSpell":{"spellId":{"spellId":9835}}}}`,
		`{"action":{"condition":` + mana + `,"castSpell":{"spellId":{"spellId":9912}}}}`,
	}
	return `{"type":"TypeAPL","priorityList":[` + strings.Join(pl, ",") + `]}`
}

func petJSON() string { b, _ := protojson.Marshal(hunterPetRef); return string(b) }

func wyldd(talents, gate string) *proto.Player {
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassDruid, Race: proto.Race_RaceWindshaperSkyborne,
		Equipment:   core.GetGearSet("../../../ui/specs/druid/balance/gear_sets", "prebis").GearSet,
		Consumables: &proto.ConsumesSpec{}, Buffs: &proto.IndividualBuffs{},
		TalentsString: talents, DistanceFromTarget: 20, ReactionTimeMs: 100,
		Rotation: core.APLRotationFromJsonString(wylddAPL(petJSON(), gate)),
	}, &proto.Player_RestorationDruid{RestorationDruid: &proto.RestorationDruid{Options: &proto.RestorationDruid_Options{ClassOptions: &proto.DruidOptions{}}}})
}

const wylddResto = "55322201--5050035103113031"

func t3scen(sc string) *proto.Encounter {
	switch sc {
	case "1 mob 60s":
		return trash(1, 60)
	case "4 mobs 45s":
		return trash(4, 45)
	case "trio boss 32k":
		e := trash(1, 300)
		e.Targets[0].Level = 62
		e.Targets[0].MinBaseDamage = 900
		e.Targets[0].Stats = make([]float64, 40)
		e.Targets[0].Stats[proto.Stat_StatHealth] = 32000
		e.UseHealth = true
		return e
	}
	e := trash(1, 90)
	e.Targets[0].Level = 62
	e.Targets[0].MinBaseDamage = 900
	return e
}

type t3 struct {
	name   string
	hunter *proto.Player
	druid  *proto.Player
	rogue  *proto.Player
}

func runT3(c t3, sc string, iters int32) *proto.RaidSimResult {
	players := []*proto.Player{c.hunter, c.druid}
	if c.rogue != nil && sc != "stress boss 90s" {
		players = append(players, c.rogue)
	}
	r := &proto.Raid{Parties: []*proto.Party{{Players: players, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
		Tanks: []*proto.UnitReference{hunterPetRef}}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: t3scen(sc), SimOptions: &proto.SimOptions{Iterations: iters, RandomSeed: 23}})
	if res.Error != nil {
		panic(c.name + ": " + res.Error.Message)
	}
	return res
}

func pr3(tag, sc string, c t3, res *proto.RaidSimResult) {
	ps := res.RaidMetrics.Parties[0].Players
	h, d := ps[0], ps[1]
	pet := h.Pets[0]
	rd, rt := 0.0, 0.0
	if len(ps) > 2 {
		rd, rt = ps[2].Dps.Avg, ps[2].Threat.Avg
	}
	fmt.Printf("%s|%s|%s|groupDPS=%.0f|petDPS=%.0f|petTPS=%.0f|petDTPS=%.0f|petDeath=%.3f|hunterDPS=%.0f|hunterRatio=%.2f|druidHPS=%.0f|druidOOM=%.1f|rogueDPS=%.0f|rogueRatio=%.2f\n",
		tag, sc, c.name, h.Dps.Avg+d.Dps.Avg+rd, pet.Dps.Avg, pet.Threat.Avg, pet.Dtps.Avg, pet.ChanceOfDeath,
		h.Dps.Avg, h.Threat.Avg/pet.Threat.Avg, d.Hps.Avg, d.SecondsOomAvg, rd, rt/pet.Threat.Avg)
}

// First look: every rogue build, with a BM hunter and a Turtle tanking, healed by Wyldd.
func TestT3First(t *testing.T) {
	for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
		for _, rb := range rogueBuilds {
			c := t3{rb.name, hawke(bmTalents, proto.HunterOptions_Turtle), wyldd(wylddResto, "50%"), shryke(rb.tal, rb.gear, rb.bs)}
			pr3("T3", sc, c, runT3(c, sc, 500))
		}
	}
}

// Hawke built to keep the pet alive: BM with Endurance Training 5, Improved Mend Pet 2, Spirit Bond 2
// and Intimidation, the rest in Marksmanship.
const bmTank = "5520001525121251-00530501"

type hk struct {
	name, tal           string
	pet                 proto.HunterOptions_PetType
	gs, na, hardy, aggr int32
	mendAt              string // pet health % that starts Mend Pet; "" for none
	holdSec             string // seconds Hawke waits before shooting; "" for none
}

// hawkeTank: the turret rotation, with Intimidation on cooldown, Mend Pet below mendAt, and an
// opening hold so the pet gets ahead on threat.
func hawkeTank(h hk) *proto.Player {
	p := hawke(h.tal, h.pet)
	o := p.GetHunter().Options.ClassOptions
	p.DistanceFromTarget = 20 // inside Mend Pet's 20 yd
	o.GreatStamina, o.NaturalArmor, o.PetHardiness, o.PetAggression = h.gs, h.na, h.hardy, h.aggr
	pet := petJSON()
	var pre []*proto.APLListItem
	if h.holdSec != "" {
		pre = append(pre, &proto.APLListItem{Action: core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[{"action":{"condition":{"cmp":{"op":"OpLt","lhs":{"currentTime":{}},"rhs":{"const":{"val":"` + h.holdSec + `s"}}}},"wait":{"duration":{"const":{"val":"0.25s"}}}}}]}`).PriorityList[0].Action})
	}
	if h.mendAt != "" {
		pre = append(pre, core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[{"action":{"condition":{"and":{"vals":[{"cmp":{"op":"OpLe","lhs":{"currentHealthPercent":{"sourceUnit":`+pet+`}},"rhs":{"const":{"val":"`+h.mendAt+`%"}}}},{"not":{"val":{"dotIsActive":{"spellId":{"spellId":13544},"targetUnit":`+pet+`}}}}]}},"channelSpell":{"spellId":{"spellId":13544},"target":`+pet+`}}}]}`).PriorityList...)
	}
	pre = append(pre, core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[{"action":{"castSpell":{"spellId":{"spellId":19577}}}}]}`).PriorityList...)
	p.Rotation.PriorityList = append(pre, p.Rotation.PriorityList...)
	return p
}

func shrykeFeint(tal, gear string, bs bool) *proto.Player {
	p := shryke(tal, gear, bs)
	feint := core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[{"action":{"castSpell":{"spellId":{"spellId":25302}}}}]}`).PriorityList
	p.Rotation.PriorityList = append(feint, p.Rotation.PriorityList...)
	return p
}

var t3scens = []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k", "stress boss 90s"}

// Pet families, Hawke's talents and the pet's training, with Shryke on Combat BS + Feint.
func TestT3Pet(t *testing.T) {
	T, B, Be, C := proto.HunterOptions_Turtle, proto.HunterOptions_Boar, proto.HunterOptions_Bear, proto.HunterOptions_Crocilisk
	cs := []hk{
		{"Turtle, preset BM, untrained", bmTalents, T, 0, 0, 0, 0, "", ""},
		{"Turtle, preset BM, GS10 NA10", bmTalents, T, 10, 10, 0, 0, "", ""},
		{"Turtle, tank BM, GS10 NA10", bmTank, T, 10, 10, 0, 0, "", ""},
		{"Turtle, tank BM, GS10 NA10, Mend 40%", bmTank, T, 10, 10, 0, 0, "40", ""},
		{"Turtle, tank BM, GS10 NA10 Hardy5, Mend 40%", bmTank, T, 10, 10, 5, 0, "40", ""},
		{"Turtle, tank BM, GS10 NA10 Aggr5, Mend 40%", bmTank, T, 10, 10, 0, 5, "40", ""},
		{"Boar, tank BM, GS10 NA10, Mend 40%", bmTank, B, 10, 10, 0, 0, "40", ""},
		{"Bear, tank BM, GS10 NA10, Mend 40%", bmTank, Be, 10, 10, 0, 0, "40", ""},
		{"Croc, tank BM, GS10 NA10, Mend 40%", bmTank, C, 10, 10, 0, 0, "40", ""},
	}
	rb := rogueBuilds[3]
	for _, sc := range t3scens {
		for _, h := range cs {
			c := t3{h.name, hawkeTank(h), wyldd(wylddResto, "50%"), shrykeFeint(rb.tal, rb.gear, rb.bs)}
			pr3("TP", sc, c, runT3(c, sc, 1000))
		}
	}
}

// Every rogue build, with and without Feint on cooldown, beside the best pet setup.
func TestT3Rogue(t *testing.T) {
	h := hk{"", bmTank, proto.HunterOptions_Turtle, 10, 10, 5, 0, "80", ""}
	for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
		for _, rb := range rogueBuilds {
			for _, f := range []bool{false, true} {
				r, n := shryke(rb.tal, rb.gear, rb.bs), rb.name
				if f {
					r, n = shrykeFeint(rb.tal, rb.gear, rb.bs), rb.name+" + Feint"
				}
				c := t3{n, hawkeTank(h), wyldd(wylddResto, "50%"), r}
				pr3("TR", sc, c, runT3(c, sc, 1000))
			}
		}
	}
}

func TestT3Debug(t *testing.T) {
	h := hk{"", bmTank, proto.HunterOptions_Turtle, 10, 10, 0, 0, "80", "3"}
	c := t3{"dbg", hawkeTank(h), wyldd(wylddResto, "50%"), nil}
	r := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{c.hunter, c.druid}, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}, Tanks: []*proto.UnitReference{hunterPetRef}}
	cs := core.ComputeStats(&proto.ComputeStatsRequest{Raid: r, Encounter: t3scen("stress boss 90s")})
	if cs.ErrorResult != "" {
		t.Fatal(cs.ErrorResult)
	}
	for i, a := range cs.RaidStats.Parties[0].Players[0].RotationStats.PriorityList {
		for _, v := range a.Validations {
			fmt.Printf("VAL %d %s\n", i, v.Validation)
		}
	}
	res := runT3(c, "stress boss 90s", 200)
	for _, a := range res.RaidMetrics.Parties[0].Players[0].Actions {
		var n int32
		var hl float64
		for _, tg := range a.Targets {
			n += tg.Casts
			hl += tg.Healing
		}
		fmt.Printf("ACT %v casts=%.1f heal=%.0f\n", a.Id, float64(n)/200, hl/200)
	}
}

// Keeping the pet up on the hard fights: Mend Pet thresholds and Wyldd never casting Wrath.
func TestT3Keep(t *testing.T) {
	rb := rogueBuilds[3]
	for _, sc := range []string{"4 mobs 45s", "stress boss 90s"} {
		for _, mend := range []string{"", "40", "60", "80"} {
			for _, gate := range []string{"50%", "101%"} {
				h := hk{"", bmTank, proto.HunterOptions_Turtle, 10, 10, 5, 0, mend, ""}
				c := t3{fmt.Sprintf("mend %s wrath-gate %s", mend, gate), hawkeTank(h), wyldd(wylddResto, gate), shrykeFeint(rb.tal, rb.gear, rb.bs)}
				res := runT3(c, sc, 1000)
				pr3("TK", sc, c, res)
				fmt.Printf("   hunterOOM=%.1f\n", res.RaidMetrics.Parties[0].Players[0].SecondsOomAvg)
			}
		}
	}
}

// Wyldd's talents, healing the trained Turtle with Mend Pet at 80% and Shryke on Combat BS + Feint.
func TestT3Wyldd(t *testing.T) {
	builds := []struct{ n, tal string }{
		{"A Resto 31 / Balance 20", wylddResto},
		{"B Resto 41 / Balance 10", "55--5053035153113051"},
		{"C Resto 41 / Feral 10 (Heart of the Wild)", "-05032-5053035153113051"},
	}
	rb := rogueBuilds[3]
	h := hk{"", bmTank, proto.HunterOptions_Turtle, 10, 10, 5, 0, "80", ""}
	for _, sc := range t3scens {
		for _, b := range builds {
			c := t3{b.n, hawkeTank(h), wyldd(b.tal, "50%"), shrykeFeint(rb.tal, rb.gear, rb.bs)}
			pr3("TW", sc, c, runT3(c, sc, 1000))
		}
	}
}
