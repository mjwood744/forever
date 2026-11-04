// Package rotations searches for the priority list that scores best for one player's build.
package rotations

import (
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wowsims/forever/assets/database"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	googleProto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	defaultIterations = 300
	maxRounds         = 15
	timeBudget        = 60 * time.Second
	// A change is kept only when it raises the goal's figure by more than this share, so the search
	// does not chase the noise left between two lists simmed on the same seeds.
	minGain = 0.003
)

// A list the search may try, with what it changes from the list it came from.
type candidate struct {
	items  []*proto.APLListItem
	change string
}

// Starts from the player's rotation and keeps the best of these changes, one per round, until none
// helps: moving an action up one place, moving it to the top, removing it, and adding a spell the
// player has and the list does not name, at the top or at the bottom. A damage-over-time spell is
// added with a condition that its dot is not already up. Every list in a search is simmed on the same
// seeds, so two lists differ only by their rotation.
func Search(req *proto.RotationSearchRequest) (result *proto.RotationSearchResult) {
	defer func() {
		if r := recover(); r != nil {
			result = &proto.RotationSearchResult{Error: &proto.ErrorOutcome{Message: fmt.Sprint(r)}}
		}
	}()
	base := req.GetRequest()
	if base == nil || len(base.GetRaid().GetParties()) == 0 || len(base.Raid.Parties[0].Players) == 0 {
		return errorResult("the request has no player to search a rotation for")
	}
	iterations := req.Iterations
	if iterations <= 0 {
		iterations = defaultIterations
	}
	seed := base.GetSimOptions().GetRandomSeed()
	if seed == 0 {
		seed = 1
	}
	player := base.Raid.Parties[0].Players[0]
	start := player.GetRotation()
	if start == nil || len(start.PriorityList) == 0 {
		return errorResult("the player's rotation has no priority list to start from")
	}

	adds, err := addableActions(base, start)
	if err != nil {
		return errorResult(err.Error())
	}

	s := &searcher{base: base, goal: req.Goal, iterations: iterations, seed: seed, rotation: start}
	visible, hidden := splitHidden(start.PriorityList)
	startScore, err := s.score(visible, hidden)
	if err != nil {
		return errorResult(err.Error())
	}

	best, bestScore := visible, startScore
	var changes []string
	deadline := time.Now().Add(timeBudget)
	for round := 0; round < maxRounds && time.Now().Before(deadline); round++ {
		cands := neighbours(best, adds)
		scores := s.scoreAll(cands, hidden)
		pick := -1
		for i, sc := range scores {
			if sc > bestScore*(1+minGain) && (pick < 0 || sc > scores[pick]) {
				pick = i
			}
		}
		if pick < 0 {
			break
		}
		best, bestScore = cands[pick].items, scores[pick]
		changes = append(changes, cands[pick].change)
	}

	out := googleProto.Clone(start).(*proto.APLRotation)
	out.Type = proto.APLRotation_TypeAPL
	out.PriorityList = append(append([]*proto.APLListItem{}, best...), hidden...)
	words := newDescriber(out).withStats(rotationStats(base, out))
	return &proto.RotationSearchResult{
		Rotation:   out,
		Score:      bestScore,
		StartScore: startScore,
		Candidates: s.simmed,
		Changes:    changes,
		Steps:      words.steps(best, 0),
		Prepull:    words.prepull(),
	}
}

func errorResult(msg string) *proto.RotationSearchResult {
	return &proto.RotationSearchResult{Error: &proto.ErrorOutcome{Message: msg}}
}

type searcher struct {
	base       *proto.RaidSimRequest
	goal       proto.RotationGoal
	iterations int32
	seed       int64
	rotation   *proto.APLRotation

	mu     sync.Mutex
	simmed int32
}

func (s *searcher) score(items, hidden []*proto.APLListItem) (float64, error) {
	req := googleProto.Clone(s.base).(*proto.RaidSimRequest)
	rot := googleProto.Clone(s.rotation).(*proto.APLRotation)
	rot.Type = proto.APLRotation_TypeAPL
	rot.PriorityList = append(append([]*proto.APLListItem{}, items...), hidden...)
	req.Raid.Parties[0].Players[0].Rotation = rot
	req.SimOptions = &proto.SimOptions{Iterations: s.iterations, RandomSeed: s.seed}
	req.Type = proto.SimType_SimTypeIndividual
	res := core.RunRaidSim(req)
	s.mu.Lock()
	s.simmed++
	s.mu.Unlock()
	if res.Error != nil {
		return 0, fmt.Errorf("%s", res.Error.Message)
	}
	m := res.GetRaidMetrics().GetParties()[0].GetPlayers()[0]
	switch s.goal {
	case proto.RotationGoal_RotationGoalTps:
		return m.GetThreat().GetAvg(), nil
	case proto.RotationGoal_RotationGoalHps:
		return m.GetHps().GetAvg(), nil
	default:
		return m.GetDps().GetAvg(), nil
	}
}

// Each candidate's figure, or -1 for one whose sim fails (such as a list the APL loop rejects).
func (s *searcher) scoreAll(cands []candidate, hidden []*proto.APLListItem) []float64 {
	scores := make([]float64, len(cands))
	work := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < max(1, runtime.GOMAXPROCS(0)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				sc, err := s.scoreSafe(cands[i].items, hidden)
				if err != nil {
					sc = -1
				}
				scores[i] = sc
			}
		}()
	}
	for i := range cands {
		work <- i
	}
	close(work)
	wg.Wait()
	return scores
}

func (s *searcher) scoreSafe(items, hidden []*proto.APLListItem) (sc float64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return s.score(items, hidden)
}

func splitHidden(items []*proto.APLListItem) (visible, hidden []*proto.APLListItem) {
	for _, it := range items {
		if it.GetHide() {
			hidden = append(hidden, it)
		} else {
			visible = append(visible, it)
		}
	}
	return visible, hidden
}

func neighbours(items []*proto.APLListItem, adds []*proto.APLListItem) []candidate {
	var out []candidate
	with := func(change string, build func([]*proto.APLListItem) []*proto.APLListItem) {
		out = append(out, candidate{items: build(append([]*proto.APLListItem{}, items...)), change: change})
	}
	for i := range items {
		i := i
		if i > 0 {
			with(fmt.Sprintf("Moved %s above %s", label(items[i]), label(items[i-1])), func(l []*proto.APLListItem) []*proto.APLListItem {
				l[i-1], l[i] = l[i], l[i-1]
				return l
			})
		}
		if i > 1 {
			with(fmt.Sprintf("Moved %s to the top", label(items[i])), func(l []*proto.APLListItem) []*proto.APLListItem {
				it := l[i]
				copy(l[1:i+1], l[:i])
				l[0] = it
				return l
			})
		}
		if len(items) > 1 {
			with(fmt.Sprintf("Removed %s", label(items[i])), func(l []*proto.APLListItem) []*proto.APLListItem {
				return append(l[:i], l[i+1:]...)
			})
		}
	}
	named := namedSpells(&proto.APLRotation{PriorityList: items})
	for _, add := range adds {
		id := castSpellID(add)
		if named[familyName(id)] {
			continue
		}
		with(fmt.Sprintf("Added %s at the top", label(add)), func(l []*proto.APLListItem) []*proto.APLListItem {
			return append([]*proto.APLListItem{add}, l...)
		})
		with(fmt.Sprintf("Added %s at the bottom", label(add)), func(l []*proto.APLListItem) []*proto.APLListItem {
			return append(l, add)
		})
	}
	return out
}

// The sim's checks of the rotation for the player, or nil when the stats can't be worked out.
func rotationStats(base *proto.RaidSimRequest, rot *proto.APLRotation) *proto.APLStats {
	raid := googleProto.Clone(base.Raid).(*proto.Raid)
	raid.Parties[0].Players[0].Rotation = rot
	stats := core.ComputeStats(&proto.ComputeStatsRequest{Raid: raid, Encounter: base.Encounter})
	if stats.ErrorResult != "" {
		return nil
	}
	return stats.GetRaidStats().GetParties()[0].GetPlayers()[0].GetRotationStats()
}

// The actions the search may add: one per spell the player has at its level and may cast on an enemy
// during the fight, its highest rank, that the starting rotation does not name.
func addableActions(base *proto.RaidSimRequest, start *proto.APLRotation) ([]*proto.APLListItem, error) {
	player := base.Raid.Parties[0].Players[0]
	stats := core.ComputeStats(&proto.ComputeStatsRequest{Raid: base.Raid, Encounter: base.Encounter, SkipRotation: true})
	if stats.ErrorResult != "" {
		return nil, fmt.Errorf("%s", stats.ErrorResult)
	}
	level := core.PlayerLevel(player)
	named := namedSpells(start)
	best := map[string]*proto.SpellStats{}
	bestLevel := map[string]int16{}
	for _, sp := range stats.GetRaidStats().GetParties()[0].GetPlayers()[0].GetMetadata().GetSpells() {
		id := sp.GetId().GetSpellId()
		if id == 0 || !sp.IsCastable || sp.PrepullOnly || sp.IsFriendly || sp.IsPotion || sp.IsConjured {
			continue
		}
		row := spelldata.Find(id)
		if row == nil || row.Name == "" || int32(row.SpellLevel) > level || named[row.Name] {
			continue
		}
		if b := best[row.Name]; b == nil || row.SpellLevel > bestLevel[row.Name] {
			best[row.Name], bestLevel[row.Name] = sp, row.SpellLevel
		}
	}
	names := make([]string, 0, len(best))
	for name := range best {
		names = append(names, name)
	}
	sort.Strings(names)
	adds := make([]*proto.APLListItem, 0, len(names))
	for _, name := range names {
		adds = append(adds, castAction(best[name]))
	}
	return adds, nil
}

func castAction(sp *proto.SpellStats) *proto.APLListItem {
	id := &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: sp.GetId().GetSpellId()}}
	action := &proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: id}}}
	if sp.HasDot {
		action.Condition = &proto.APLValue{Value: &proto.APLValue_Not{Not: &proto.APLValueNot{Val: &proto.APLValue{
			Value: &proto.APLValue_DotIsActive{DotIsActive: &proto.APLValueDotIsActive{SpellId: id}},
		}}}}
	}
	return &proto.APLListItem{Action: action}
}

func castSpellID(it *proto.APLListItem) int32 {
	return it.GetAction().GetCastSpell().GetSpellId().GetSpellId()
}

// A spell's name from the spell store, or from the database's spell names for a spell whose row has none.
func familyName(id int32) string {
	if s := spelldata.Find(id); s != nil && s.Name != "" {
		return s.Name
	}
	if name := dbNames().spells[id]; name != "" {
		return name
	}
	return fmt.Sprint(id)
}

var camelWords = regexp.MustCompile(`([a-z])([A-Z])`)

type names struct{ spells, items map[int32]string }

var dbNames = sync.OnceValue(func() names {
	db := database.Load()
	n := names{spells: map[int32]string{}, items: map[int32]string{}}
	for _, s := range db.SpellIcons {
		n.spells[s.Id] = s.Name
	}
	for _, it := range db.Items {
		n.items[it.Id] = it.Name
	}
	for _, it := range db.ItemIcons {
		if n.items[it.Id] == "" {
			n.items[it.Id] = it.Name
		}
	}
	return n
})

// The names of the spells a rotation names anywhere: in actions, conditions and groups.
func namedSpells(rot *proto.APLRotation) map[string]bool {
	named := map[string]bool{}
	walk(rot.ProtoReflect(), func(id int32) { named[familyName(id)] = true })
	return named
}

func walk(m protoreflect.Message, visit func(int32)) {
	walkIDs(m, func(id *proto.ActionID) {
		if sid := id.GetSpellId(); sid != 0 {
			visit(sid)
		}
	})
}

func walkIDs(m protoreflect.Message, visit func(*proto.ActionID)) {
	if id, ok := m.Interface().(*proto.ActionID); ok {
		visit(id)
		return
	}
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() != protoreflect.MessageKind {
			return true
		}
		switch {
		case fd.IsList():
			for i := 0; i < v.List().Len(); i++ {
				walkIDs(v.List().Get(i).Message(), visit)
			}
		case fd.IsMap():
		default:
			walkIDs(v.Message(), visit)
		}
		return true
	})
}

// The name of the first spell or item a message names, or "" when it names none.
func firstName(m protoreflect.Message) (name string) {
	walkIDs(m, func(id *proto.ActionID) {
		if name != "" {
			return
		}
		if sid := id.GetSpellId(); sid != 0 {
			name = familyName(sid)
		} else if iid := id.GetItemId(); iid != 0 {
			if name = dbNames().items[iid]; name == "" {
				name = fmt.Sprintf("item %d", iid)
			}
		} else if other := id.GetOtherId(); other != proto.OtherAction_OtherActionNone {
			// OtherActionResourcePotion reads "Resource Potion".
			name = strings.TrimSpace(camelWords.ReplaceAllString(strings.TrimPrefix(other.String(), "OtherAction"), "$1 $2"))
		}
	})
	return name
}

// A short name for an action, as the page lists it.
func label(it *proto.APLListItem) string {
	return (&describer{}).action(it.GetAction())
}

// An action's spell or item, or its kind when it names neither.
func kindLabel(a *proto.APLAction) string {
	m := a.ProtoReflect()
	fd := m.WhichOneof(m.Descriptor().Oneofs().ByName("action"))
	if fd == nil {
		return "an empty action"
	}
	if name := firstName(m.Get(fd).Message()); name != "" {
		return name
	}
	return strings.ReplaceAll(string(fd.Name()), "_", " ")
}
