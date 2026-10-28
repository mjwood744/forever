package mysim

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Macro checks for Trio 3 per levelling bracket. Character level stays 60 in the sim,
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
func castSeqOf(name, cond string, steps ...string) []string {
	done := fmt.Sprintf(`{"sequenceIsComplete":{"sequenceName":"%s"}}`, name)
	return []string{
		item(fmt.Sprintf(`{"resetSequence":{"sequenceName":"%s"}}`, name), done),
		item(fmt.Sprintf(`{"sequence":{"name":"%s","actions":[%s]}}`, name, strings.Join(steps, ",")), cond),
	}
}

func castSeq(name, cond string, ids ...int) []string {
	steps := make([]string, len(ids))
	for i, id := range ids {
		steps[i] = cs(id)
	}
	return castSeqOf(name, cond, steps...)
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

func cmpV(op, lhs, v string) string {
	return `{"cmp":{"op":"` + op + `","lhs":` + lhs + `,"rhs":{"const":{"val":"` + v + `"}}}}`
}

const multi3 = `{"cmp":{"op":"OpGt","lhs":{"numberTargets":{}},"rhs":{"const":{"val":"1"}}}}`

type b3 struct {
	lvl                  int
	hawke, wyldd, shryke string
}

var brackets3 = []b3{
	{30, "3520001523", "--50500151031", "-320303202015"},
	{42, "352000152502125", "02--5050035103113031", "-32030320201515231"},
	{60, bmTank, wylddResto, "00532310101-32030320201515231-002"},
}

func trioOf(b b3, hAPL, wAPL, sAPL string) t3 {
	h := hawkeTank(hk{"", b.hawke, proto.HunterOptions_Turtle, 10, 10, 5, 0, "80", ""})
	if hAPL != "" {
		h.Rotation = core.APLRotationFromJsonString(hAPL)
	}
	w := wyldd(b.wyldd, "50%")
	if wAPL != "" {
		w.Rotation = core.APLRotationFromJsonString(wAPL)
	}
	s := shrykeFeint(b.shryke, "p1_bscombat", true)
	if sAPL != "" {
		s.Rotation = core.APLRotationFromJsonString(sAPL)
	}
	return t3{"", h, w, s}
}

func p3(tag string, b b3, sc, name string, res *proto.RaidSimResult) {
	ps := res.RaidMetrics.Parties[0].Players
	h, d, r := ps[0], ps[1], ps[2]
	pet := h.Pets[0]
	fmt.Printf("%s|L%d|%s|%s|petDeath=%.3f|petTPS=%.0f|hunterDPS=%.0f|hunterRatio=%.2f|druidHPS=%.0f|druidOOM=%.1f|druidDPS=%.0f|rogueDPS=%.0f|rogueRatio=%.2f|groupDPS=%.0f\n",
		tag, b.lvl, sc, name, pet.ChanceOfDeath, pet.Threat.Avg, h.Dps.Avg, h.Threat.Avg/pet.Threat.Avg, d.Hps.Avg, d.SecondsOomAvg, d.Dps.Avg,
		r.Dps.Avg, r.Threat.Avg/pet.Threat.Avg, h.Dps.Avg+d.Dps.Avg+r.Dps.Avg)
}

func TestShrykeKeys(t *testing.T) {
	const feint, backstab, snd, evis, garrote, stealth, adrenaline, flurry = 25302, 11281, 6774, 11300, 11290, 1787, 13750, 13877
	cp5 := cmpV("OpGe", `{"currentComboPoints":{}}`, "5")
	stealthed := fmt.Sprintf(`{"auraIsActive":{"auraId":{"spellId":%d}}}`, stealth)
	pp := []string{pre(stealth, "-1s")}
	for _, b := range brackets3 {
		burst := []string{item(`{"autocastOtherCooldowns":{}}`, "")}
		if b.lvl >= 42 {
			burst = append(burst, lines("", adrenaline)...)
		}
		if b.lvl >= 31 {
			burst = append(burst, lines(multi3, flurry)...)
		}
		builder := append(lines("", feint), append(lines(stealthed, garrote), lines("", backstab)...)...)
		vs := []struct{ n, a string }{
			{"Reference: the sim's rogue order + Feint", ""},
			{"Draft: finisher SnD, Eviscerate, Eviscerate at 5 points", apl(pp, burst, castSeq("F", cp5, snd, evis, evis), builder)},
			{"Finisher SnD, Eviscerate at 5 points", apl(pp, burst, castSeq("F", cp5, snd, evis), builder)},
			{"Finisher SnD, Eviscerate x3 at 5 points", apl(pp, burst, castSeq("F", cp5, snd, evis, evis, evis), builder)},
			{"Eviscerate only at 5 points (no Slice and Dice)", apl(pp, burst, lines(cp5, evis), builder)},
		}
		for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
			for _, x := range vs {
				c := trioOf(b, "", "", x.a)
				p3("SH", b, sc, x.n, runT3(c, sc, 1000))
			}
		}
	}
}

func hawkeAPL(key1 []string) string {
	pet := petJSON()
	mend := `{"action":{"condition":{"and":{"vals":[{"cmp":{"op":"OpLe","lhs":{"currentHealthPercent":{"sourceUnit":` + pet + `}},"rhs":{"const":{"val":"80%"}}}},{"not":{"val":{"dotIsActive":{"spellId":{"spellId":13544},"targetUnit":` + pet + `}}}}]}},"channelSpell":{"spellId":{"spellId":13544},"target":` + pet + `}}}`
	pp := []string{pre(14325, "-22s"), pre(25296, "-20s")}
	return apl(pp, []string{mend}, lines("", 19577), []string{item(`{"autocastOtherCooldowns":{}}`, "")}, key1)
}

func TestHawkeKeys(t *testing.T) {
	const aimed, serpent, arcane, hawk = 20904, 25295, 14287, 1293527
	for _, b := range brackets3 {
		vs := []struct{ n, a string }{
			{"Reference: the sim's order (Aimed, Serpent kept up, Arcane)", ""},
			{"Draft: castsequence Aimed, Serpent, Arcane", hawkeAPL(castSeq("H", "", aimed, serpent, arcane))},
			{"/cast Aimed Shot, /cast Arcane Shot, then castsequence Serpent", hawkeAPL(append(lines("", aimed, arcane), castSeq("S", "", serpent)...))},
			{"castsequence Aimed, Serpent, Arcane, Arcane", hawkeAPL(castSeq("H", "", aimed, serpent, arcane, arcane))},
		}
		if b.lvl >= 46 {
			vs = append(vs, struct{ n, a string }{"Draft from 46: castsequence Aimed, Serpent, Arcane, Summon Hawk", hawkeAPL(castSeq("H", "", aimed, serpent, arcane, hawk))})
		}
		for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
			for _, x := range vs {
				c := trioOf(b, x.a, "", "")
				p3("HK", b, sc, x.n, runT3(c, sc, 1000))
			}
		}
	}
}

func TestWylddKeys(t *testing.T) {
	const regrowth, rejuv, swiftmend, touch, wildGrowth, ns, moonfire, wrath = 9858, 25299, 18562, 25297, 1238215, 17116, 9835, 9912
	pet := petJSON()
	hp := `{"currentHealthPercent":{"sourceUnit":` + pet + `}}`
	le := func(v string) string { return cmpV("OpLe", hp, v) }
	on := func(id int, cond string) string { return item(csAt(id, pet), cond) }
	mana := cmpV("OpGe", `{"currentManaPercent":{}}`, "50%")
	dmg := []string{
		item(cs(moonfire), `{"and":{"vals":[`+mana+`,{"not":{"val":{"dotIsActive":{"spellId":{"spellId":9835}}}}}]}}`),
		item(cs(wrath), mana),
	}
	steady := func(cond string, ids ...int) []string {
		steps := make([]string, len(ids))
		for i, id := range ids {
			steps[i] = csAt(id, pet)
		}
		return castSeqOf("W", cond, steps...)
	}
	for _, b := range brackets3 {
		ohNo := []string{}
		if b.lvl >= 33 {
			ohNo = append(ohNo, on(ns, le("45%")))
		}
		ohNo = append(ohNo, on(swiftmend, le("45%")), on(touch, le("45%")))
		wg := []string{}
		if b.lvl >= 40 {
			wg = []string{on(wildGrowth, le("80%"))}
		}
		vs := []struct{ n, a string }{
			{"Reference: the sim's heal order", ""},
			{"Draft: Oh no below 45%, Steady (Regrowth, Rejuvenation) below 90%", apl(nil, ohNo, wg, steady(le("90%"), regrowth, rejuv), dmg)},
			{"Steady as Rejuvenation, Regrowth below 90%", apl(nil, ohNo, wg, steady(le("90%"), rejuv, regrowth), dmg)},
			{"Steady (Regrowth, Rejuvenation) below 75%", apl(nil, ohNo, wg, steady(le("75%"), regrowth, rejuv), dmg)},
			{"Oh no without Nature's Swiftness", apl(nil, []string{on(swiftmend, le("45%")), on(touch, le("45%"))}, wg, steady(le("90%"), regrowth, rejuv), dmg)},
		}
		for _, sc := range []string{"4 mobs 45s", "trio boss 32k", "stress boss 90s"} {
			for _, x := range vs {
				c := trioOf(b, "", x.a, "")
				if sc == "stress boss 90s" {
					c.rogue = nil
				}
				res := runT3(c, sc, 1000)
				if c.rogue == nil {
					ps := res.RaidMetrics.Parties[0].Players
					fmt.Printf("WD|L%d|%s|%s|petDeath=%.3f|druidHPS=%.0f|druidOOM=%.1f\n", b.lvl, sc, x.n, ps[0].Pets[0].ChanceOfDeath, ps[1].Hps.Avg, ps[1].SecondsOomAvg)
					continue
				}
				p3("WD", b, sc, x.n, res)
			}
		}
	}
}

func TestShrykeBreakdown(t *testing.T) {
	const feint, backstab, snd, evis, garrote, stealth = 25302, 11281, 6774, 11300, 11290, 1787
	cp5 := cmpV("OpGe", `{"currentComboPoints":{}}`, "5")
	stealthed := fmt.Sprintf(`{"auraIsActive":{"auraId":{"spellId":%d}}}`, stealth)
	builder := append(lines("", feint), append(lines(stealthed, garrote), lines("", backstab)...)...)
	b := brackets3[2]
	for _, x := range []struct{ n, a string }{
		{"ref", ""},
		{"draft", apl([]string{pre(stealth, "-1s")}, castSeq("F", cp5, snd, evis, evis), builder)},
		{"evis only", apl([]string{pre(stealth, "-1s")}, lines(cp5, evis), builder)},
		{"seq evis evis", apl([]string{pre(stealth, "-1s")}, castSeq("F", cp5, evis, evis), builder)},
		{"seq snd snd", apl([]string{pre(stealth, "-1s")}, castSeq("F", cp5, snd, snd), builder)},
	} {
		c := trioOf(b, "", "", x.a)
		res := runT3(c, "trio boss 32k", 300)
		r := res.RaidMetrics.Parties[0].Players[2]
		fmt.Printf("== %s rogueDPS=%.0f\n", x.n, r.Dps.Avg)
		for _, a := range r.Actions {
			n, d := int32(0), 0.0
			for _, tg := range a.Targets {
				n += tg.Casts
				d += tg.Damage
			}
			if n > 0 && (a.Id.GetSpellId() == snd || a.Id.GetSpellId() == evis) {
				fmt.Printf("  %v casts=%.1f dmg=%.0f\n", a.Id, float64(n)/300, d/300)
			}
		}
	}
}

func TestShrykeThresholds(t *testing.T) {
	const feint, backstab, snd, evis, garrote, stealth, adrenaline, flurry = 25302, 11281, 6774, 11300, 11290, 1787, 13750, 13877
	cp := func(n string) string { return cmpV("OpGe", `{"currentComboPoints":{}}`, n) }
	stealthed := fmt.Sprintf(`{"auraIsActive":{"auraId":{"spellId":%d}}}`, stealth)
	pp := []string{pre(stealth, "-1s")}
	for _, b := range brackets3 {
		burst := []string{item(`{"autocastOtherCooldowns":{}}`, "")}
		if b.lvl >= 42 {
			burst = append(burst, lines("", adrenaline)...)
		}
		if b.lvl >= 31 {
			burst = append(burst, lines(multi3, flurry)...)
		}
		builder := append(lines("", feint), append(lines(stealthed, garrote), lines("", backstab)...)...)
		vs := []struct{ n, a string }{
			{"Finisher SnD, Evis, Evis at 5 points (draft)", apl(pp, burst, castSeq("F", cp("5"), snd, evis, evis), builder)},
			{"Finisher SnD, Evis, Evis at 4 points", apl(pp, burst, castSeq("F", cp("4"), snd, evis, evis), builder)},
			{"Finisher SnD, Evis, Evis at 3 points", apl(pp, burst, castSeq("F", cp("3"), snd, evis, evis), builder)},
			{"Finisher SnD, Evis x3 at 4 points", apl(pp, burst, castSeq("F", cp("4"), snd, evis, evis, evis), builder)},
			{"Eviscerate only at 4 points", apl(pp, burst, lines(cp("4"), evis), builder)},
			{"Eviscerate only at 3 points", apl(pp, burst, lines(cp("3"), evis), builder)},
		}
		for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
			for _, x := range vs {
				c := trioOf(b, "", "", x.a)
				p3("SHT", b, sc, x.n, runT3(c, sc, 1000))
			}
		}
	}
}

func TestWylddSteady(t *testing.T) {
	const regrowth, rejuv, swiftmend, touch, wildGrowth, moonfire, wrath = 9858, 25299, 18562, 25297, 1238215, 9835, 9912
	pet := petJSON()
	hp := `{"currentHealthPercent":{"sourceUnit":` + pet + `}}`
	le := func(v string) string { return cmpV("OpLe", hp, v) }
	on := func(id int, cond string) string { return item(csAt(id, pet), cond) }
	mana := cmpV("OpGe", `{"currentManaPercent":{}}`, "50%")
	dmg := []string{
		item(cs(moonfire), `{"and":{"vals":[`+mana+`,{"not":{"val":{"dotIsActive":{"spellId":{"spellId":9835}}}}}]}}`),
		item(cs(wrath), mana),
	}
	steady := func(cond string, ids ...int) []string {
		steps := make([]string, len(ids))
		for i, id := range ids {
			steps[i] = csAt(id, pet)
		}
		return castSeqOf("W", cond, steps...)
	}
	for _, b := range brackets3 {
		ohNo := []string{on(swiftmend, le("45%")), on(touch, le("45%"))}
		wg := []string{}
		if b.lvl >= 40 {
			wg = []string{on(wildGrowth, le("80%"))}
		}
		vs := []struct{ n, a string }{
			{"Reference: the sim's heal order", ""},
			{"Draft: Steady (Regrowth, Rejuvenation) below 90%", apl(nil, ohNo, wg, steady(le("90%"), regrowth, rejuv), dmg)},
			{"Steady (Regrowth, Rejuvenation) below 80%", apl(nil, ohNo, wg, steady(le("80%"), regrowth, rejuv), dmg)},
			{"Steady (Rejuvenation, Healing Touch, Healing Touch) below 80%", apl(nil, ohNo, wg, steady(le("80%"), rejuv, touch, touch), dmg)},
			{"Steady (Regrowth, Rejuvenation, Healing Touch, Healing Touch) below 80%", apl(nil, ohNo, wg, steady(le("80%"), regrowth, rejuv, touch, touch), dmg)},
			{"Steady (Rejuvenation, Regrowth, Healing Touch) below 80%", apl(nil, ohNo, wg, steady(le("80%"), rejuv, regrowth, touch), dmg)},
		}
		for _, sc := range []string{"4 mobs 45s", "trio boss 32k", "stress boss 90s"} {
			for _, x := range vs {
				c := trioOf(b, "", x.a, "")
				if sc == "stress boss 90s" {
					c.rogue = nil
				}
				res := runT3(c, sc, 1000)
				ps := res.RaidMetrics.Parties[0].Players
				g := ps[0].Dps.Avg + ps[1].Dps.Avg
				if len(ps) > 2 {
					g += ps[2].Dps.Avg
				}
				fmt.Printf("WS|L%d|%s|%s|petDeath=%.3f|druidOOM=%.1f|druidDPS=%.0f|groupDPS=%.0f\n", b.lvl, sc, x.n, ps[0].Pets[0].ChanceOfDeath, ps[1].SecondsOomAvg, ps[1].Dps.Avg, g)
			}
		}
	}
}
