package mysim

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Macro checks for Trio 2 per levelling bracket. Character level stays 60 in the sim,
// so each bracket only limits talents and which spells the keys use.

func cs(id int) string { return fmt.Sprintf(`{"castSpell":{"spellId":{"spellId":%d}}}`, id) }

func csAt(id int, target string) string {
	return fmt.Sprintf(`{"castSpell":{"spellId":{"spellId":%d},"target":%s}}`, id, target)
}

func item(action, cond string) string {
	if cond == "" {
		return `{"action":` + action + `}`
	}
	return `{"action":` + strings.Replace(action, "{", `{"condition":`+cond+`,`, 1) + `}`
}

// castsequence: reset once the last step is cast, otherwise wait on the current step.
func castSeq(name, cond string, ids ...int) []string {
	steps := make([]string, len(ids))
	for i, id := range ids {
		steps[i] = cs(id)
	}
	done := fmt.Sprintf(`{"sequenceIsComplete":{"sequenceName":"%s"}}`, name)
	return []string{
		item(fmt.Sprintf(`{"resetSequence":{"sequenceName":"%s"}}`, name), done),
		item(fmt.Sprintf(`{"sequence":{"name":"%s","actions":[%s]}}`, name, strings.Join(steps, ",")), cond),
	}
}

func lines(cond string, ids ...int) []string {
	var out []string
	for _, id := range ids {
		out = append(out, item(cs(id), cond))
	}
	return out
}

func apl(prepull []string, prio ...[]string) string {
	var all []string
	for _, p := range prio {
		all = append(all, p...)
	}
	return `{"type":"TypeAPL","prepullActions":[` + strings.Join(prepull, ",") + `],"priorityList":[` + strings.Join(all, ",") + `]}`
}

func pre(id int, at string) string {
	return fmt.Sprintf(`{"action":{"castSpell":{"spellId":{"spellId":%d}}},"doAtValue":{"const":{"val":"%s"}}}`, id, at)
}

func cmpGe(lhs, v string) string {
	return `{"cmp":{"op":"OpGe","lhs":` + lhs + `,"rhs":{"const":{"val":"` + v + `"}}}}`
}

func cmpLe(lhs, v string) string {
	return `{"cmp":{"op":"OpLe","lhs":` + lhs + `,"rhs":{"const":{"val":"` + v + `"}}}}`
}

const (
	rage  = `{"currentRage":{}}`
	mana  = `{"currentManaPercent":{}}`
	multi = `{"cmp":{"op":"OpGt","lhs":{"numberTargets":{}},"rhs":{"const":{"val":"1"}}}}`
)

type b2 struct {
	lvl             int
	skarr, wy, lock string
}

var brackets2 = []b2{
	{24, "--0505200003", "--533000301", "03-03050031"},
	{30, "--050532020300001", "--5530003014", "03-03050032012002"},
	{42, "001--050533120300001351", "2--5530003015013131", "03-030500320122030135"},
	{60, "35300003-05-050533120300001351", "5503331--5530003015013131", "03-0305003201220301351-0050055002"},
}

func runB2(b b2, sc, skAPL, wyAPL, lockAPL string) []*proto.UnitMetrics {
	sk := warrior(wb{"s", b.skarr, true})
	if skAPL != "" {
		sk.Rotation = core.APLRotationFromJsonString(skAPL)
	}
	w := wyther(b.wy, "50%", true)
	if wyAPL != "" {
		w.Rotation = core.APLRotationFromJsonString(wyAPL)
	}
	m := malyceBuild("demo_fire_brand")
	m.TalentsString = b.lock
	if lockAPL != "" {
		m.Rotation = core.APLRotationFromJsonString(lockAPL)
	}
	r := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{w, sk, m}, Buffs: &proto.PartyBuffs{BloodPact: true}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
		Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: bEnc2(sc), SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 31}})
	if res.Error != nil {
		panic(res.Error.Message)
	}
	return res.RaidMetrics.Parties[0].Players
}

func bEnc2(sc string) *proto.Encounter {
	if sc == "1 mob 120s" {
		return enc(1, true, 120)
	}
	return scen2(sc)
}

func p2(tag string, b b2, sc, name string, ps []*proto.UnitMetrics) {
	fmt.Printf("%s|L%d|%s|%s|skTPS=%.0f|lockRatio=%.2f|skDTPS=%.0f|death=%.3f|skDPS=%.0f|wyDPS=%.0f|wyOOM=%.1f|lockDPS=%.0f|lockOOM=%.1f|groupDPS=%.0f\n",
		tag, b.lvl, sc, name, ps[1].Threat.Avg, ps[2].Threat.Avg/ps[1].Threat.Avg, ps[1].Dtps.Avg, ps[1].ChanceOfDeath, ps[1].Dps.Avg,
		ps[0].Dps.Avg, ps[0].SecondsOomAvg, ps[2].Dps.Avg, ps[2].SecondsOomAvg, ps[0].Dps.Avg+ps[1].Dps.Avg+ps[2].Dps.Avg)
}

func TestSkarrKeys(t *testing.T) {
	const shieldBlock, shieldSlam, revenge, sunder, heroic, bloodFury = 2565, 23925, 25288, 11597, 25286, 20572
	const thunderClap, demo, cleave, battleShout, bloodrage = 11581, 11556, 20569, 25289, 2687
	pp := []string{pre(battleShout, "-1.5s"), pre(bloodrage, "-1.5s")}
	dump := lines(cmpGe(rage, "40"), heroic)
	for _, b := range brackets2[1:] {
		k1 := []int{bloodFury, shieldBlock, revenge, sunder}
		if b.lvl >= 40 {
			k1 = []int{bloodFury, shieldBlock, shieldSlam, revenge, sunder}
		}
		noBlock := []int{bloodFury}
		for _, id := range k1[2:] {
			noBlock = append(noBlock, id)
		}
		pack := append(lines(multi, cleave), lines(multi, thunderClap, demo)...)
		packRage := append(lines(`{"and":{"vals":[`+multi+`,`+cmpGe(rage, "40")+`]}}`, cleave), lines(multi, thunderClap, demo)...)
		vs := []struct{ n, a string }{
			{"Reference: the sim's Prot order", ""},
			{"Key 1 only", apl(pp, lines("", k1...))},
			{"Key 1 + rage dump at 40", apl(pp, dump, lines("", k1...))},
			{"Key 1 + rage dump + Pack key on packs (Cleave every press)", apl(pp, pack, dump, lines("", k1...))},
			{"Key 1 + rage dump + Pack key on packs (Cleave at 40 rage)", apl(pp, packRage, dump, lines("", k1...))},
			{"Shield Block moved to the rage dump key, + Pack key", apl(pp, packRage, lines(cmpGe(rage, "40"), shieldBlock, heroic), lines("", noBlock...))},
		}
		for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
			for _, x := range vs {
				p2("SK", b, sc, x.n, runB2(b, sc, x.a, "", ""))
			}
		}
	}
}

func TestMalyceKeys(t *testing.T) {
	const searing, immolate, corruption, shadowBolt, agony, lifeTap = 17923, 25309, 25311, 25307, 11713, 11689
	tap := lines(cmpLe(mana, "30%"), lifeTap)
	petPull := `{"action":{"petAttack":{}},"doAtValue":{"const":{"val":"-1s"}}}`
	for _, b := range brackets2 {
		type v struct {
			n   string
			ids []int
		}
		var vs []v
		if b.lvl < 29 {
			vs = []v{
				{"Draft: Immolate, Corruption, Agony, Shadow Bolt x2", []int{immolate, corruption, agony, shadowBolt, shadowBolt}},
				{"Immolate, Corruption, Agony, Shadow Bolt x4", []int{immolate, corruption, agony, shadowBolt, shadowBolt, shadowBolt, shadowBolt}},
				{"Immolate, Corruption, Agony, Searing Pain x2", []int{immolate, corruption, agony, searing, searing}},
				{"Immolate, Corruption, Agony, Searing Pain x4", []int{immolate, corruption, agony, searing, searing, searing, searing}},
			}
		} else {
			vs = []v{
				{"Draft: Searing Pain, Immolate, Corruption, Searing Pain x3", []int{searing, immolate, corruption, searing, searing, searing}},
				{"Searing Pain, Immolate, Corruption, Searing Pain x2", []int{searing, immolate, corruption, searing, searing}},
				{"Searing Pain, Immolate, Corruption, Searing Pain x5", []int{searing, immolate, corruption, searing, searing, searing, searing, searing}},
				{"Searing Pain, Immolate, Corruption, Agony, Searing Pain x3", []int{searing, immolate, corruption, agony, searing, searing, searing}},
				{"Immolate, Corruption, Searing Pain x4", []int{immolate, corruption, searing, searing, searing, searing}},
				{"Shadow Bolt version: Immolate, Corruption, Agony, Shadow Bolt x3", []int{immolate, corruption, agony, shadowBolt, shadowBolt, shadowBolt}},
			}
		}
		for _, sc := range []string{"1 mob 60s", "1 mob 120s", "4 mobs 45s", "trio boss 32k"} {
			for _, x := range vs {
				a := apl([]string{petPull}, tap, castSeq("L", "", x.ids...), lines("", lifeTap))
				p2("ML", b, sc, x.n, runB2(b, sc, "", "", a))
			}
			p2("ML", b, sc, "Reference: the sim's Demo rotation", runB2(b, sc, "", "", ""))
		}
	}
}

func TestWytherKeys(t *testing.T) {
	const bolt, flameShock, lavaBurst = 15208, 29228, 1326047
	full := shamanAPL("50%", true)
	// the heal lines and the totems from the sim's own Wyther, with the attack key in place of its damage lines
	cut := strings.Index(full, `,{"action":{"condition":{"and":{"vals":[{"cmp":{"op":"OpGe","lhs":{"currentManaPercent"`)
	head := full[:cut]
	gate := cmpGe(mana, "50%")
	for _, b := range brackets2[1:] {
		vs := []struct {
			n   string
			ids []int
		}{
			{"Bolt, Flame Shock, Bolt, Bolt (draft before 40)", []int{bolt, flameShock, bolt, bolt}},
			{"Bolt, Flame Shock, Lava Burst, Bolt, Bolt (draft from 40)", []int{bolt, flameShock, lavaBurst, bolt, bolt}},
			{"Flame Shock, Lava Burst, Bolt x3", []int{flameShock, lavaBurst, bolt, bolt, bolt}},
			{"Bolt, Flame Shock, Lava Burst, Bolt x3", []int{bolt, flameShock, lavaBurst, bolt, bolt, bolt}},
			{"Flame Shock, Bolt x3", []int{flameShock, bolt, bolt, bolt}},
		}
		for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
			for _, x := range vs {
				seq := castSeq("W", gate, x.ids...)
				a := head + "," + strings.Join(seq, ",") + "]}"
				p2("WY", b, sc, x.n, runB2(b, sc, "", a, ""))
			}
			p2("WY", b, sc, "Reference: the sim's order (Flame Shock kept up, Bolt)", runB2(b, sc, "", "", ""))
		}
	}
}

func TestSkarrBossDebuffs(t *testing.T) {
	const shieldBlock, shieldSlam, revenge, sunder, heroic, bloodFury = 2565, 23925, 25288, 11597, 25286, 20572
	const thunderClap, demo, battleShout, bloodrage = 11581, 11556, 25289, 2687
	pp := []string{pre(battleShout, "-1.5s"), pre(bloodrage, "-1.5s")}
	dump := lines(cmpGe(rage, "40"), heroic)
	off := func(id int) string {
		return fmt.Sprintf(`{"auraIsInactive":{"auraId":{"spellId":%d},"sourceUnit":{"type":"CurrentTarget"}}}`, id)
	}
	for _, b := range []b2{brackets2[1], brackets2[3]} {
		k1 := []int{bloodFury, shieldBlock, revenge, sunder}
		if b.lvl >= 40 {
			k1 = []int{bloodFury, shieldBlock, shieldSlam, revenge, sunder}
		}
		vs := []struct{ n, a string }{
			{"Reference: the sim's Prot order", ""},
			{"Key 1 + rage dump", apl(pp, dump, lines("", k1...))},
			{"Key 1 + rage dump + Thunder Clap and Demo Shout kept up", apl(pp, []string{item(cs(thunderClap), off(thunderClap)), item(cs(demo), off(demo))}, dump, lines("", k1...))},
		}
		for _, sc := range []string{"1 mob 60s", "trio boss 32k", "stress boss 90s"} {
			for _, x := range vs {
				p2("SKB", b, sc, x.n, runB2(b, sc, x.a, "", ""))
			}
		}
	}
}

func TestMalyceBreakdown(t *testing.T) {
	const searing, immolate, corruption, lifeTap = 17923, 25309, 25311, 11689
	petPull := `{"action":{"petAttack":{}},"doAtValue":{"const":{"val":"-1s"}}}`
	b := brackets2[1]
	curse := lines(`{"not":{"val":{"auraIsActive":{"auraId":{"spellId":1311680},"sourceUnit":{"type":"CurrentTarget"}}}}}`, 1311680)
	draft := apl([]string{petPull}, lines(cmpLe(mana, "30%"), lifeTap), []string{item(`{"autocastOtherCooldowns":{}}`, "")}, curse, castSeq("L", "", immolate, corruption, 11713, 25307, 25307, 25307), lines("", lifeTap))
	for _, x := range []struct{ n, a string }{{"ref", ""}, {"draft", draft}} {
		sk := warrior(wb{"s", b.skarr, true})
		w := wyther(b.wy, "50%", true)
		m := malyceBuild("demo_fire_brand")
		m.TalentsString = b.lock
		if x.a != "" {
			m.Rotation = core.APLRotationFromJsonString(x.a)
		}
		r := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{w, sk, m}, Buffs: &proto.PartyBuffs{BloodPact: true}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
			Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
		res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: bEnc2("trio boss 32k"), SimOptions: &proto.SimOptions{Iterations: 300, RandomSeed: 31}})
		pm := res.RaidMetrics.Parties[0].Players[2]
		fmt.Printf("== %s lockDPS=%.0f dur=%.1f\n", x.n, pm.Dps.Avg, res.AvgIterationDuration)
		show := func(who string, acts []*proto.ActionMetrics) {
			for _, a := range acts {
				c, d := int32(0), 0.0
				for _, tg := range a.Targets {
					c += tg.Casts
					d += tg.Damage
				}
				if d > 0 || c > 0 {
					fmt.Printf("  %s %v casts=%.1f dmg=%.0f\n", who, a.Id, float64(c)/300, d/300)
				}
			}
		}
		show("lock", pm.Actions)
		for _, p := range pm.Pets {
			fmt.Printf("  pet %s dps=%.0f\n", p.Name, p.Dps.Avg)
		}
	}
}

func TestMalyceKeys2(t *testing.T) {
	const searing, immolate, corruption, shadowBolt, agony, lifeTap, coe, soulFire = 17923, 25309, 25311, 25307, 11713, 11689, 1311680, 17924
	petPull := `{"action":{"petAttack":{}},"doAtValue":{"const":{"val":"-1s"}}}`
	tap := lines(cmpLe(mana, "30%"), lifeTap)
	curse := lines(fmt.Sprintf(`{"not":{"val":{"auraIsActive":{"auraId":{"spellId":%d},"sourceUnit":{"type":"CurrentTarget"}}}}}`, coe), coe)
	cds := []string{item(`{"autocastOtherCooldowns":{}}`, "")}
	for _, b := range brackets2[1:] {
		vs := []struct {
			n   string
			pre []string
			ids []int
		}{
			{"Draft: Searing Pain, Immolate, Corruption, Searing Pain x3", nil, []int{searing, immolate, corruption, searing, searing, searing}},
			{"Searing Pain, Immolate, Corruption, Searing Pain x5", nil, []int{searing, immolate, corruption, searing, searing, searing, searing, searing}},
			{"Searing Pain, Immolate, Corruption, Agony, Searing Pain x3", nil, []int{searing, immolate, corruption, agony, searing, searing, searing}},
			{"Shadow Bolt version: Immolate, Corruption, Agony, Shadow Bolt x3", nil, []int{immolate, corruption, agony, shadowBolt, shadowBolt, shadowBolt}},
			{"Draft + Soul Fire cast before the pull", []string{`{"action":{"castSpell":{"spellId":{"spellId":17924}}},"doAtValue":{"const":{"val":"-4s"}}}`}, []int{searing, immolate, corruption, searing, searing, searing}},
		}
		for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
			for _, x := range vs {
				a := apl(append([]string{petPull}, x.pre...), tap, cds, curse, castSeq("L", "", x.ids...), lines("", lifeTap))
				p2("ML2", b, sc, x.n+" (+ curse, trinkets)", runB2(b, sc, "", "", a))
			}
			p2("ML2", b, sc, "Reference: the sim's Demo rotation", runB2(b, sc, "", "", ""))
		}
	}
}

func TestMalyceSwitch(t *testing.T) {
	const searing, immolate, corruption, shadowBolt, agony, lifeTap, coe = 17923, 25309, 25311, 25307, 11713, 11689, 1311680
	petPull := `{"action":{"petAttack":{}},"doAtValue":{"const":{"val":"-1s"}}}`
	tap := lines(cmpLe(mana, "30%"), lifeTap)
	curse := lines(fmt.Sprintf(`{"not":{"val":{"auraIsActive":{"auraId":{"spellId":%d},"sourceUnit":{"type":"CurrentTarget"}}}}}`, coe), coe)
	cds := []string{item(`{"autocastOtherCooldowns":{}}`, "")}
	for _, b := range []b2{
		{42, brackets2[2].skarr, brackets2[2].wy, "03-030500320122030135"},
		{48, brackets2[3].skarr, brackets2[3].wy, "03-0305003201220301351-005"},
		{51, brackets2[3].skarr, brackets2[3].wy, "03-0305003201220301351-005003"},
		{54, brackets2[3].skarr, brackets2[3].wy, "03-0305003201220301351-0050051"},
		{57, brackets2[3].skarr, brackets2[3].wy, "03-0305003201220301351-0050054"},
		brackets2[3],
	} {
		for _, sc := range []string{"1 mob 60s", "trio boss 32k"} {
			for _, x := range []struct {
				n   string
				ids []int
			}{
				{"Searing Pain loop (SP, Immolate, Corruption, SP x5)", []int{searing, immolate, corruption, searing, searing, searing, searing, searing}},
				{"Shadow Bolt loop (Immolate, Corruption, Agony, SB x3)", []int{immolate, corruption, agony, shadowBolt, shadowBolt, shadowBolt}},
			} {
				a := apl([]string{petPull}, tap, cds, curse, castSeq("L", "", x.ids...), lines("", lifeTap))
				p2("MS", b, sc, x.n, runB2(b, sc, "", "", a))
			}
		}
	}
}
