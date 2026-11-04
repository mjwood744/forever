package rotations

import (
	"fmt"
	"strings"

	"github.com/wowsims/forever/sim/core/proto"
)

// Puts a rotation's shown actions into words for the page: the ability each one uses and, for one with a
// condition, when it runs. A value the describer has no words for is named after its kind and the spell it
// reads, e.g. "Arcane Blast spell cpm", so every condition still reads as something.
type describer struct {
	rot  *proto.APLRotation
	vars map[string]*proto.APLValue
	// Actions this character can never run, by group (-1 for the priority list) and place; nil lists all.
	dead map[[2]int]bool
	// The prepull actions this character can never run, by place.
	deadPrepull map[int]bool
}

func newDescriber(rot *proto.APLRotation) *describer {
	d := &describer{rot: rot, vars: map[string]*proto.APLValue{}}
	for _, v := range rot.GetValueVariables() {
		d.vars[v.GetName()] = v.GetValue()
	}
	return d
}

func (d *describer) steps(items []*proto.APLListItem, depth int) []*proto.RotationStep {
	return d.stepsIn(items, depth, -1)
}

func (d *describer) stepsIn(items []*proto.APLListItem, depth, group int) []*proto.RotationStep {
	var out []*proto.RotationStep
	for i, it := range items {
		if it.GetHide() || d.dead[[2]int{group, i}] {
			continue
		}
		// An action whose condition is always false, one the rotation switches off, never runs.
		st := d.step(it.GetAction(), depth)
		// A group left with nothing this character can run is left out with them.
		if st.When != "false" && (it.GetAction().GetGroupReference() == nil || depth > 0 || len(st.Steps) > 0 || d.dead == nil) {
			out = append(out, st)
		}
	}
	return out
}

func (d *describer) step(a *proto.APLAction, depth int) *proto.RotationStep {
	st := &proto.RotationStep{Ability: d.action(a), Conditional: a.GetCondition() != nil}
	if c := a.GetCondition(); c != nil {
		if st.When = d.value(c); st.When == "true" {
			st.When, st.Conditional = "", false
		}
	}
	// A group runs its own list in place; its actions are listed under it, one level deep.
	if g := a.GetGroupReference(); g != nil && depth == 0 {
		for gi, grp := range d.rot.GetGroups() {
			if grp.GetName() == g.GetGroupName() {
				for _, v := range grp.GetVariables() {
					if _, ok := d.vars[v.GetName()]; !ok {
						d.vars[v.GetName()] = v.GetValue()
					}
				}
				st.Steps = d.stepsIn(grp.GetActions(), depth+1, gi)
			}
		}
	}
	return st
}

func (d *describer) prepull() []*proto.RotationStep {
	var out []*proto.RotationStep
	for i, p := range d.rot.GetPrepullActions() {
		if p.GetHide() || d.deadPrepull[i] {
			continue
		}
		st := &proto.RotationStep{Ability: d.action(p.GetAction())}
		if at := p.GetDoAtValue().GetConst().GetVal(); at != "" {
			st.When = strings.TrimPrefix(at, "-") + " before the pull"
		}
		out = append(out, st)
	}
	return out
}

// What the sim said about an action that means this character never runs it: a spell or item it doesn't
// have, a cooldown group with nothing in it, a pet or swap set it lacks.
var neverRuns = []string{"does not know", "will not cast any spells", "never cast", "picks no potion", "No swap set", "Unknown item swap", "No pet", "No melee"}

func cannotRun(vs []*proto.APLValidation) bool {
	for _, v := range vs {
		if v.GetLogLevel() == proto.LogLevel_Error {
			return true
		}
		for _, w := range neverRuns {
			if v.GetLogLevel() == proto.LogLevel_Warning && strings.Contains(v.GetValidation(), w) {
				return true
			}
		}
	}
	return false
}

// Leaves out of the words every action the sim's checks of the rotation say this character can't run.
func (d *describer) withStats(stats *proto.APLStats) *describer {
	if stats == nil {
		return d
	}
	d.dead, d.deadPrepull = map[[2]int]bool{}, map[int]bool{}
	for i, a := range stats.GetPriorityList() {
		d.dead[[2]int{-1, i}] = cannotRun(a.GetValidations())
	}
	for g, grp := range stats.GetGroups() {
		for i, a := range grp.GetActions() {
			d.dead[[2]int{g, i}] = cannotRun(a.GetValidations())
		}
	}
	for i, a := range stats.GetPrepullActions() {
		d.deadPrepull[i] = cannotRun(a.GetValidations())
	}
	return d
}

func (d *describer) action(a *proto.APLAction) string {
	switch {
	case a.GetGroupReference() != nil:
		return a.GetGroupReference().GetGroupName()
	case a.GetMultidot() != nil:
		return spellName(a.GetMultidot().GetSpellId()) + " on every target"
	case a.GetCastAllStatBuffCooldowns() != nil:
		return "Every cooldown that raises your stats"
	case a.GetAutocastOtherCooldowns() != nil:
		return "Any other cooldown that is ready"
	case a.GetWait() != nil:
		return "Wait " + d.value(a.GetWait().GetDuration())
	case a.GetWaitUntil() != nil:
		return "Wait until " + d.value(a.GetWaitUntil().GetCondition())
	case a.GetCastWarlockAssignedCurse() != nil:
		return "Your assigned curse"
	case a.GetPetAttack() != nil:
		return "Send your pet in"
	case a.GetStartAttack() != nil:
		return "Start auto attacking"
	case a.GetCancelAura() != nil:
		return "Cancel " + spellName(a.GetCancelAura().GetAuraId())
	}
	for _, seq := range [][]*proto.APLAction{a.GetSequence().GetActions(), a.GetStrictSequence().GetActions()} {
		if len(seq) > 0 {
			names := make([]string, len(seq))
			for i, s := range seq {
				names[i] = d.action(s)
			}
			return strings.Join(names, ", then ")
		}
	}
	return kindLabel(a)
}

var compareWords = map[proto.APLValueCompare_ComparisonOperator]string{
	proto.APLValueCompare_OpEq: "is",
	proto.APLValueCompare_OpNe: "is not",
	proto.APLValueCompare_OpLt: "is below",
	proto.APLValueCompare_OpLe: "is at most",
	proto.APLValueCompare_OpGt: "is above",
	proto.APLValueCompare_OpGe: "is at least",
}

var mathWords = map[proto.APLValueMath_MathOperator]string{
	proto.APLValueMath_OpAdd: "plus",
	proto.APLValueMath_OpSub: "minus",
	proto.APLValueMath_OpMul: "times",
	proto.APLValueMath_OpDiv: "divided by",
}

// Words for the values that read the same whatever their spell or aura.
var plainValues = map[string]string{
	"current_time":                     "the fight time",
	"current_time_percent":             "the share of the fight gone",
	"remaining_time":                   "the time left in the fight",
	"remaining_time_percent":           "the share of the fight left",
	"is_execute_phase":                 "the target is in execute range",
	"number_targets":                   "the number of enemies",
	"current_health":                   "your health",
	"current_health_percent":           "your health",
	"current_mana":                     "your mana",
	"current_mana_percent":             "your mana",
	"current_rage":                     "your rage",
	"current_energy":                   "your energy",
	"current_combo_points":             "your combo points",
	"max_mana":                         "your maximum mana",
	"gcd_is_ready":                     "the global cooldown is ready",
	"gcd_time_to_ready":                "the time until the global cooldown is ready",
	"unit_is_moving":                   "you are moving",
	"warlock_assigned_curse_is_active": "your assigned curse is on the target",
	"warlock_assigned_curse_is_active_on_all_targets": "your assigned curse is on every target",
	"warlock_bane_is_active_on_all_targets":           "a Bane is on every target",
	"auto_time_to_next":                               "the time until your next swing",
	"auto_time_since_last":                            "the time since your last swing",
	"auto_swing_time":                                 "your swing time",
}

// Words for the values that read one spell or aura, with %s for its name.
var spellValues = map[string]string{
	"spell_is_known":               "you know %s",
	"spell_can_cast":               "%s can be cast",
	"spell_is_ready":               "%s is off cooldown",
	"spell_time_to_ready":          "the time until %s is off cooldown",
	"spell_cast_time":              "the cast time of %s",
	"spell_is_casting":             "you are casting %s",
	"spell_is_channeling":          "you are channeling %s",
	"spell_expected_dpet":          "the damage per second of casting %s",
	"spell_travel_time":            "the travel time of %s",
	"spell_cpm":                    "how often %s is cast",
	"spell_current_cost":           "the cost of %s",
	"aura_is_known":                "you have %s",
	"aura_is_active":               "%s is active",
	"aura_is_inactive":             "%s is not active",
	"aura_remaining_time":          "the time left on %s",
	"aura_num_stacks":              "the number of %s stacks",
	"aura_should_refresh":          "%s needs refreshing",
	"aura_icd_is_ready":            "%s can proc again",
	"dot_is_active":                "%s is on the target",
	"dot_remaining_time":           "the time left on %s",
	"dot_is_active_on_all_targets": "%s is on every target",
	"dot_lowest_remaining_time":    "the shortest time left on %s",
}

// Words for a value that is true or false, put in the negative.
var notValues = map[string]string{
	"aura_is_active":                   "%s is not active",
	"aura_is_inactive":                 "%s is active",
	"dot_is_active":                    "%s is not on the target",
	"spell_is_ready":                   "%s is on cooldown",
	"spell_is_channeling":              "you are not channeling %s",
	"spell_is_casting":                 "you are not casting %s",
	"aura_should_refresh":              "%s does not need refreshing",
	"is_execute_phase":                 "the target is not in execute range",
	"unit_is_moving":                   "you are standing still",
	"warlock_assigned_curse_is_active": "your assigned curse is not on the target",
	"dot_is_active_on_all_targets":     "%s is missing from a target",
}

func (d *describer) value(v *proto.APLValue) string {
	if v == nil {
		return ""
	}
	switch {
	case v.GetConst() != nil:
		return v.GetConst().GetVal()
	case v.GetAnd() != nil:
		return d.join(v.GetAnd().GetVals(), " and ", "true", "false")
	case v.GetOr() != nil:
		return d.join(v.GetOr().GetVals(), " or ", "false", "true")
	case v.GetNot() != nil:
		inner := d.resolve(v.GetNot().GetVal())
		kind, name := kindAndSpell(inner)
		if f, ok := notValues[kind]; ok {
			return withName(f, name)
		}
		switch text := d.value(inner); text {
		case "true":
			return "false"
		case "false":
			return "true"
		default:
			return "not (" + text + ")"
		}
	case v.GetCmp() != nil:
		c := v.GetCmp()
		lhs, op := d.value(c.GetLhs()), compareWords[c.GetOp()]
		if strings.HasSuffix(lhs, "points") {
			op = "are" + strings.TrimPrefix(op, "is")
		}
		return lhs + " " + op + " " + d.value(c.GetRhs())
	case v.GetMath() != nil:
		m := v.GetMath()
		return d.value(m.GetLhs()) + " " + mathWords[m.GetOp()] + " " + d.value(m.GetRhs())
	case v.GetMax() != nil:
		return "the largest of " + d.join(v.GetMax().GetVals(), ", ", "", "")
	case v.GetMin() != nil:
		return "the smallest of " + d.join(v.GetMin().GetVals(), ", ", "", "")
	case v.GetActionGroupUsed() != nil:
		return v.GetActionGroupUsed().GetName() + " has been used"
	case v.GetMultipleCdUsages() != nil:
		return "using it now still fits every use in the fight"
	case v.GetVariableRef() != nil:
		name := v.GetVariableRef().GetName()
		// A short variable reads better inline than under its name.
		if inner, ok := d.vars[name]; ok {
			if text := d.value(inner); len(text) <= 40 {
				return text
			}
		}
		return name
	}
	kind, name := kindAndSpell(v)
	if s, ok := plainValues[kind]; ok {
		return s
	}
	if f, ok := spellValues[kind]; ok {
		return withName(f, name)
	}
	text := strings.ReplaceAll(kind, "_", " ")
	if name != "" {
		text = name + " " + text
	}
	return text
}

// A variable's own value, so "not" can read it in the negative; any other value as it is.
func (d *describer) resolve(v *proto.APLValue) *proto.APLValue {
	if ref := v.GetVariableRef(); ref != nil {
		if inner, ok := d.vars[ref.GetName()]; ok && len(d.value(inner)) <= 40 {
			return inner
		}
	}
	return v
}

func withName(format, name string) string {
	if !strings.Contains(format, "%s") {
		return format
	}
	return fmt.Sprintf(format, name)
}

// A part of an "and" or "or" that itself joins parts is put in brackets. A part that reads as skip (true in
// an "and", false in an "or") is left out, and one that reads as decide makes the whole that value: a
// rotation's on and off switches are variables set to true or false.
func (d *describer) join(vals []*proto.APLValue, sep, skip, decide string) string {
	var parts []string
	for _, p := range vals {
		text := d.value(p)
		if skip != "" && text == skip {
			continue
		}
		if decide != "" && text == decide {
			return decide
		}
		if (p.GetAnd() != nil || p.GetOr() != nil) && (strings.Contains(text, " and ") || strings.Contains(text, " or ")) {
			text = "(" + text + ")"
		}
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return skip
	}
	return strings.Join(parts, sep)
}

// The kind a value is, as its field name, and the name of the first spell it reads.
func kindAndSpell(v *proto.APLValue) (kind, name string) {
	m := v.ProtoReflect()
	fd := m.WhichOneof(m.Descriptor().Oneofs().ByName("value"))
	if fd == nil {
		return "", ""
	}
	return string(fd.Name()), firstName(m.Get(fd).Message())
}

func spellName(id *proto.ActionID) string {
	if id == nil {
		return "a spell"
	}
	if name := firstName(id.ProtoReflect()); name != "" {
		return name
	}
	return "a spell"
}
