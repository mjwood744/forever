package mysim

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Macro checks for Trio 1 per levelling bracket. Character level stays 60 in the sim,
// so each bracket only limits talents and which spells the keys use.

func cs(id int) string { return fmt.Sprintf(`{"castSpell":{"spellId":{"spellId":%d}}}`, id) }

func csT(id int, target string) string {
	return fmt.Sprintf(`{"castSpell":{"spellId":{"spellId":%d},"target":{"type":"%s"}}}`, id, target)
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

func apl(prepull []string, prio ...[]string) string {
	var all []string
	for _, p := range prio {
		all = append(all, p...)
	}
	return `{"type":"TypeAPL","prepullActions":[` + strings.Join(prepull, ",") + `],"priorityList":[` + strings.Join(all, ",") + `]}`
}

func one(s string) []string { return []string{s} }

const (
	frostbolt, fireball, fireBlast, arcaneBlast = 10181, 10151, 10199, 1239700
	arcaneWrath, pom, arcanePower, evocation    = 1326082, 12043, 12042, 12051
	shoot                                       = 5019
)

func manaAtLeast(p string) string {
	return fmt.Sprintf(`{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"%s"}}}}`, p)
}

func ready(id int) string { return fmt.Sprintf(`{"spellIsReady":{"spellId":{"spellId":%d}}}`, id) }

type brk struct {
	lvl             int
	rel, nyx, ashyn string
}

var brackets = []brk{
	{18, "-55000", "504", "0302004"},
	{30, "-5500012301301", "505005020030001", "0302205003100311"},
	{42, "-5500013301301551", "505005020330011521", "0512205003100311531"},
	{60, "5-5500013301301551-0522103", "505025021330011531-00505103", "0552205003100311531-03-0533"},
}

func trio(b brk, nyxAPL, ashAPL, relAPL string) *proto.Raid {
	tank := player(build{"r", b.rel, true})
	if relAPL != "" {
		tank.Rotation = core.APLRotationFromJsonString(relAPL)
	}
	pr := priestPlayer(b.nyx, "1316991")
	if nyxAPL == "" {
		nyxAPL = nyxxAPL("50%")
	}
	pr.Rotation = core.APLRotationFromJsonString(nyxAPL)
	m := ashyn()
	m.TalentsString = b.ashyn
	m.DistanceFromTarget = 20 // Fire Blast's range; at 25 yards a castsequence holding it never moves on
	if ashAPL != "" {
		m.Rotation = core.APLRotationFromJsonString(ashAPL)
	}
	party := &proto.Party{Players: []*proto.Player{pr, tank, m}, Buffs: &proto.PartyBuffs{DevotionAura: true}}
	return &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
		Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
}

func bEnc(sc string) *proto.Encounter {
	switch sc {
	case "1 mob 60s":
		return encD(1, 60, 60, 350)
	case "1 mob 120s":
		return encD(1, 120, 60, 350)
	case "4 mobs 45s":
		return encD(4, 45, 60, 350)
	case "boss 90s":
		return encD(1, 90, 62, 900)
	}
	return scen("trio boss 32k")
}

func runB(r *proto.Raid, sc string) []*proto.UnitMetrics {
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: bEnc(sc), SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 77}})
	if res.Error != nil {
		panic(res.Error.Message)
	}
	return res.RaidMetrics.Parties[0].Players
}

func pline(tag string, b brk, sc, name string, ps []*proto.UnitMetrics) {
	fmt.Printf("%s|L%d|%s|%s|ashynDPS=%.1f|ashynOOM=%.1f|ashynThreat/Rellyk=%.2f|rellykTPS=%.1f|rellykDTPS=%.1f|death=%.3f|nyxxDPS=%.1f|nyxxHPS=%.1f|nyxxOOM=%.1f|groupDPS=%.1f\n",
		tag, b.lvl, sc, name, ps[2].Dps.Avg, ps[2].SecondsOomAvg, ps[2].Threat.Avg/ps[1].Threat.Avg, ps[1].Threat.Avg, ps[1].Dtps.Avg, ps[1].ChanceOfDeath,
		ps[0].Dps.Avg, ps[0].Hps.Avg, ps[0].SecondsOomAvg, ps[0].Dps.Avg+ps[1].Dps.Avg+ps[2].Dps.Avg)
}

func TestAshynKeys(t *testing.T) {
	evo := one(item(`{"channelSpell":{"spellId":{"spellId":12051},"interruptIf":{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"95%"}}}}}}`,
		`{"cmp":{"op":"OpLe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"10%"}}}}`))
	wand := one(item(csT(shoot, "Target"), ""))
	burst30 := []string{item(cs(pom), ready(arcaneWrath)), item(cs(arcaneWrath), "")}
	burst42 := append([]string{item(cs(arcanePower), ready(arcaneWrath))}, burst30...)
	smart, _ := os.ReadFile("../../../ui/specs/mage/dps/apls/arcaneFrostFill.apl.json")
	type v struct {
		name string
		apl  string
	}
	for _, b := range brackets {
		var vs []v
		if b.lvl < 25 {
			vs = []v{
				{"Frostbolt, Fire Blast, Frostbolt x4", apl(nil, castSeq("A", "", frostbolt, fireBlast, frostbolt, frostbolt, frostbolt, frostbolt), wand)},
				{"Frostbolt, Fire Blast, Frostbolt x3", apl(nil, castSeq("A", "", frostbolt, fireBlast, frostbolt, frostbolt, frostbolt), wand)},
				{"Frostbolt, Fire Blast, Frostbolt x2", apl(nil, castSeq("A", "", frostbolt, fireBlast, frostbolt, frostbolt), wand)},
				{"Fireball, Fire Blast, Fireball x3", apl(nil, castSeq("A", "", fireball, fireBlast, fireball, fireball, fireball), wand)},
				{"Frostbolt only", apl(nil, one(item(cs(frostbolt), "")), wand)},
				{"Frostbolt, Fire Blast whenever ready", apl(nil, one(item(cs(fireBlast), "")), one(item(cs(frostbolt), "")), wand)},
				{"Fireball only", apl(nil, one(item(cs(fireball), "")), wand)},
			}
		} else {
			burst := burst30
			if b.lvl >= 41 {
				burst = burst42
			}
			for _, n := range []int{2, 3, 4} {
				ids := []int{}
				for i := 0; i < n; i++ {
					ids = append(ids, arcaneBlast)
				}
				ids = append(ids, frostbolt, fireBlast, frostbolt)
				name := fmt.Sprintf("Arcane Blast x%d, Frostbolt, Fire Blast, Frostbolt", n)
				vs = append(vs, v{name + " (no Burst)", apl(nil, evo, castSeq("A", "", ids...), wand)})
				vs = append(vs, v{name + " + Burst on cooldown", apl(nil, evo, burst, castSeq("A", "", ids...), wand)})
			}
			vs = append(vs, v{"Arcane Blast x4, Frostbolt x2 (no Fire Blast) + Burst on cooldown", apl(nil, evo, burst, castSeq("A", "", arcaneBlast, arcaneBlast, arcaneBlast, arcaneBlast, frostbolt, frostbolt), wand)})
			vs = append(vs, v{"Frostbolt, Fire Blast, Frostbolt x3 + Burst on cooldown", apl(nil, evo, burst, castSeq("A", "", frostbolt, fireBlast, frostbolt, frostbolt, frostbolt), wand)})
			vs = append(vs, v{"Reference: the sim's own Arcane rotation", addShoot(string(smart))})
		}
		for _, sc := range []string{"1 mob 60s", "1 mob 120s", "trio boss 32k"} {
			for _, x := range vs {
				pline("ASH", b, sc, x.name, runB(trio(b, "", x.apl, ""), sc))
			}
		}
	}
}

func TestAshynBreakdown(t *testing.T) {
	b := brackets[1]
	if os.Getenv("LVL") == "60" {
		b = brackets[3]
	}
	smart, _ := os.ReadFile("../../../ui/specs/mage/dps/apls/arcaneFrostFill.apl.json")
	evo := one(item(`{"channelSpell":{"spellId":{"spellId":12051},"interruptIf":{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"95%"}}}}}}`,
		`{"cmp":{"op":"OpLe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"10%"}}}}`))
	burst := []string{item(cs(pom), ready(arcaneWrath)), item(cs(arcaneWrath), "")}
	for _, x := range []struct{ n, a string }{
		{"smart", addShoot(string(smart))},
		{"AB3 loop + burst", apl(nil, evo, burst, castSeq("A", "", arcaneBlast, arcaneBlast, arcaneBlast, frostbolt, fireBlast, frostbolt), one(item(csT(shoot, "Target"), "")))},
	} {
		r := trio(b, "", x.a, "")
		res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: bEnc("trio boss 32k"), SimOptions: &proto.SimOptions{Iterations: 500, RandomSeed: 77}})
		m := res.RaidMetrics.Parties[0].Players[2]
		fmt.Printf("== %s dps=%.1f dur=%.1f\n", x.n, m.Dps.Avg, res.AvgIterationDuration)
		for _, a := range m.Actions {
			c, d := int32(0), 0.0
			for _, tg := range a.Targets {
				c += tg.Casts
				d += tg.Damage
			}
			if c > 0 || d > 0 {
				fmt.Printf("  %v casts=%.1f dmg/it=%.0f\n", a.Id, float64(c)/500, d/500)
			}
		}
	}
}

func TestAshynKeys2(t *testing.T) {
	const missiles, barrage = 10212, 1270842
	evo := one(item(`{"channelSpell":{"spellId":{"spellId":12051},"interruptIf":{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"95%"}}}}}}`,
		`{"cmp":{"op":"OpLe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"10%"}}}}`))
	wand := one(item(csT(shoot, "Target"), ""))
	proc := one(item(fmt.Sprintf(`{"channelSpell":{"spellId":{"spellId":%d}}}`, missiles), fmt.Sprintf(`{"auraIsActive":{"auraId":{"spellId":%d}}}`, barrage)))
	missLoop := func(n int) []string {
		ids := []int{}
		for i := 0; i < n; i++ {
			ids = append(ids, arcaneBlast)
		}
		return castSeq("A", "", append(ids, missiles)...)
	}
	smart, _ := os.ReadFile("../../../ui/specs/mage/dps/apls/arcaneFrostFill.apl.json")
	for _, b := range brackets[1:] {
		burst := []string{item(`{"autocastOtherCooldowns":{}}`, ready(arcaneWrath)), item(cs(pom), ready(arcaneWrath)), item(cs(arcaneWrath), "")}
		if b.lvl >= 41 {
			burst = append([]string{item(cs(arcanePower), ready(arcaneWrath))}, burst...)
		}
		draft := castSeq("A", "", arcaneBlast, arcaneBlast, frostbolt, fireBlast, frostbolt)
		ab3 := castSeq("A", "", arcaneBlast, arcaneBlast, arcaneBlast, frostbolt, fireBlast, frostbolt)
		spam := one(item(cs(arcaneBlast), ""))
		vs := []struct{ n, a string }{
			{"Draft: AB x2, Frostbolt, Fire Blast, Frostbolt + Burst", apl(nil, evo, burst, draft, wand)},
			{"AB x3, Frostbolt, Fire Blast, Frostbolt + Burst", apl(nil, evo, burst, ab3, wand)},
			{"AB spam + Burst", apl(nil, evo, burst, spam, wand)},
			{"AB x4, Missiles + Burst", apl(nil, evo, burst, missLoop(4), wand)},
			{"AB x3, Missiles + Burst", apl(nil, evo, burst, missLoop(3), wand)},
			{"Draft + Burst + Missiles key when it glows", apl(nil, evo, burst, proc, draft, wand)},
			{"AB x3 loop + Burst + Missiles key when it glows", apl(nil, evo, burst, proc, ab3, wand)},
			{"AB spam + Burst + Missiles key when it glows", apl(nil, evo, burst, proc, spam, wand)},
			{"Reference: the sim's own Arcane rotation", addShoot(string(smart))},
		}
		for _, sc := range []string{"1 mob 60s", "1 mob 120s", "trio boss 32k"} {
			for _, x := range vs {
				pline("ASH2", b, sc, x.n, runB(trio(b, "", x.a, ""), sc))
			}
		}
	}
}

func healSection() []string {
	var m map[string]any
	_ = json.Unmarshal([]byte(strings.ReplaceAll(healAPL, "PENHEAL", "1316991")), &m)
	var out []string
	for _, e := range m["priorityList"].([]any)[:5] {
		js, _ := json.Marshal(e)
		out = append(out, string(js))
	}
	return out
}

func healPrepull() []string {
	return []string{`{"action":{"castSpell":{"spellId":{"spellId":10952}}},"doAtValue":{"const":{"val":"-10s"}}}`}
}

func TestNyxxKeys(t *testing.T) {
	const smite, swp, mindBlast, holyFire = 10934, 10894, 10947, 15261
	wand := one(item(csT(shoot, "Target"), ""))
	gate := manaAtLeast("50%")
	vs := []struct {
		n   string
		ids []int
	}{
		{"Smite, SW:P, Smite x3", []int{smite, swp, smite, smite, smite}},
		{"Smite, SW:P, Mind Blast, Smite, Smite", []int{smite, swp, mindBlast, smite, smite}},
		{"Holy Fire, SW:P, Mind Blast, Smite, Smite", []int{holyFire, swp, mindBlast, smite, smite}},
		{"Holy Fire, SW:P, Smite x3", []int{holyFire, swp, smite, smite, smite}},
		{"Holy Fire, SW:P, Smite x4", []int{holyFire, swp, smite, smite, smite, smite}},
	}
	for _, b := range brackets {
		for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
			for _, x := range vs {
				a := apl(healPrepull(), healSection(), castSeq("N", gate, x.ids...), wand)
				pline("NYX", b, sc, x.n, runB(trio(b, a, "", ""), sc))
			}
			pline("NYX", b, sc, "Reference: the sim's damage order", runB(trio(b, "", "", ""), sc))
		}
	}
}

func TestNyxxGreaterHeal(t *testing.T) {
	b := brackets[3]
	for _, sc := range []string{"4 mobs 45s", "trio boss 32k", "boss 90s"} {
		for _, r := range []struct {
			n  string
			id int
		}{{"Heal R4", 6064}, {"Greater Heal R1", 2060}, {"Greater Heal R2", 10963}, {"Greater Heal R3", 10964}, {"Greater Heal R4", 10965}, {"Greater Heal R5", 25314}} {
			a := strings.Replace(nyxxAPL("50%"), "25314", fmt.Sprint(r.id), 1)
			pline("GH", b, sc, r.n, runB(trio(b, a, "", ""), sc))
		}
	}
}

func TestRellykKeys(t *testing.T) {
	const judgement, holyShield, holyStrike, consecration, hammer, swiftJ = 20271, 20928, 10333, 20924, 24239, 0
	data, _ := os.ReadFile("../../../ui/specs/paladin/protection/apls/default.apl.json")
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	var pre []string
	pre = append(pre, `{"action":{"castSpell":{"spellId":{"spellId":25780}}},"doAtValue":{"const":{"val":"-20s"}}}`)
	for _, e := range m["prepullActions"].([]any) {
		js, _ := json.Marshal(e)
		pre = append(pre, string(js))
	}
	sealJS, _ := json.Marshal(m["priorityList"].([]any)[3])
	seal := one(string(sealJS))
	keys := func(ids ...int) []string {
		var out []string
		for _, id := range ids {
			out = append(out, item(cs(id), ""))
		}
		return out
	}
	multi := `{"cmp":{"op":"OpGt","lhs":{"numberTargets":{}},"rhs":{"const":{"val":"1"}}}}`
	for _, b := range brackets[1:] {
		key1 := []int{judgement, holyStrike}
		pack := []int{judgement, consecration, holyStrike}
		if b.lvl >= 40 {
			key1 = []int{judgement, holyShield, holyStrike}
			pack = []int{judgement, consecration, holyShield, holyStrike}
		}
		if b.lvl >= 44 {
			key1 = append(key1, hammer)
		}
		packOnMulti := []string{}
		for _, id := range pack {
			packOnMulti = append(packOnMulti, item(cs(id), multi))
		}
		vs := []struct{ n, a string }{
			{"Reference: the sim's Prot order", ""},
			{"Key 1 only", apl(pre, seal, keys(key1...))},
			{"Key 1, Pack key on packs", apl(pre, seal, packOnMulti, keys(key1...))},
			{"Key 1 + Consecration last, Pack key on packs", apl(pre, seal, packOnMulti, keys(key1...), keys(consecration))},
			{"Holy Shield first: key 1 reordered", apl(pre, seal, packOnMulti, keys(append([]int{holyShield}, key1...)...))},
		}
		for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k", "boss 90s"} {
			for _, x := range vs {
				if b.lvl < 40 && strings.HasPrefix(x.n, "Holy Shield") {
					continue
				}
				pline("REL", b, sc, x.n, runB(trio(b, "", "", x.a), sc))
			}
		}
	}
}

func TestRellykKeys2(t *testing.T) {
	const judgement, swiftJ, holyShield, holyStrike, consecration, exorcism = 20271, 1310994, 20928, 10333, 20924, 10314
	data, _ := os.ReadFile("../../../ui/specs/paladin/protection/apls/default.apl.json")
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	pre := []string{`{"action":{"castSpell":{"spellId":{"spellId":25780}}},"doAtValue":{"const":{"val":"-20s"}}}`}
	for _, e := range m["prepullActions"].([]any) {
		js, _ := json.Marshal(e)
		pre = append(pre, string(js))
	}
	sealJS, _ := json.Marshal(m["priorityList"].([]any)[3])
	seal := string(sealJS)
	build := func(cds bool, ids ...int) string {
		out := []string{}
		if cds {
			out = append(out, item(`{"autocastOtherCooldowns":{}}`, ""))
		}
		out = append(out, seal)
		for _, id := range ids {
			out = append(out, item(cs(id), ""))
		}
		return apl(pre, out)
	}
	for _, b := range brackets[1:] {
		hs := []int{}
		if b.lvl >= 40 {
			hs = []int{holyShield}
		}
		cat := func(parts ...[]int) []int {
			var o []int
			for _, p := range parts {
				o = append(o, p...)
			}
			return o
		}
		vs := []struct{ n, a string }{
			{"Reference: the sim's Prot order", ""},
			{"Draft key 1: Judgement, (Holy Shield), Holy Strike", build(false, cat([]int{judgement}, hs, []int{holyStrike})...)},
			{"+ Swift Judgement", build(false, cat([]int{judgement, swiftJ}, hs, []int{holyStrike})...)},
			{"+ Consecration (Pack key used everywhere)", build(false, cat([]int{judgement, consecration}, hs, []int{holyStrike})...)},
			{"+ Swift Judgement + Consecration", build(false, cat([]int{judgement, swiftJ, consecration}, hs, []int{holyStrike})...)},
			{"+ Swift Judgement + Consecration + Exorcism last", build(false, cat([]int{judgement, swiftJ, consecration}, hs, []int{holyStrike, exorcism})...)},
			{"+ all that + trinkets/racial on cooldown", build(true, cat([]int{judgement, swiftJ, consecration}, hs, []int{holyStrike, exorcism})...)},
		}
		for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k", "boss 90s"} {
			for _, x := range vs {
				ps := runB(trio(b, "", "", x.a), sc)
				fmt.Printf("REL2|L%d|%s|%s|rellykTPS=%.0f|ashynThreat/Rellyk=%.2f|rellykDTPS=%.0f|death=%.3f|rellykOOM=%.1f|rellykDPS=%.0f|groupDPS=%.0f\n", b.lvl, sc, x.n,
					ps[1].Threat.Avg, ps[2].Threat.Avg/ps[1].Threat.Avg, ps[1].Dtps.Avg, ps[1].ChanceOfDeath, ps[1].SecondsOomAvg, ps[1].Dps.Avg, ps[0].Dps.Avg+ps[1].Dps.Avg+ps[2].Dps.Avg)
			}
		}
	}
}

func TestRellykCrusader(t *testing.T) {
	data, _ := os.ReadFile("../../../ui/specs/paladin/protection/apls/default.apl.json")
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	pl := m["priorityList"].([]any)
	js := func(i int) string { b, _ := json.Marshal(pl[i]); return string(b) }
	rf := `{"action":{"castSpell":{"spellId":{"spellId":25780}}},"doAtValue":{"const":{"val":"-20s"}}}`
	pp := m["prepullActions"].([]any)
	aura, _ := json.Marshal(pp[0])
	cru, _ := json.Marshal(pp[1])
	b := brackets[3]
	vs := []struct{ n, a string }{
		{"Reference", ""},
		{"Reference order, no Crusader before the pull", apl([]string{rf, string(aura)}, []string{js(0), js(1), js(2), js(3), js(5), js(7), js(8), js(9)})},
		{"Seal first, Crusader before the pull", apl([]string{rf, string(aura), string(cru)}, []string{js(0), js(3), js(1), js(2), js(5), js(7), js(8), js(9)})},
	}
	for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
		for _, x := range vs {
			ps := runB(trio(b, "", "", x.a), sc)
			fmt.Printf("CRU|%s|%s|rellykTPS=%.0f|ashynThreat/Rellyk=%.2f|rellykDTPS=%.0f|groupDPS=%.0f\n", sc, x.n, ps[1].Threat.Avg, ps[2].Threat.Avg/ps[1].Threat.Avg, ps[1].Dtps.Avg, ps[0].Dps.Avg+ps[1].Dps.Avg+ps[2].Dps.Avg)
		}
	}
	fmt.Println(string(aura), string(cru))
}
