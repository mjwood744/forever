// Package opener tells what each player casts in the first seconds of a fight, from one simmed fight.
package opener

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	googleProto "google.golang.org/protobuf/proto"
)

// One cast in the fight.
type Cast struct {
	// Seconds from the pull; a cast before the pull is below 0.
	At     float64 `json:"at"`
	Player string  `json:"player"`
	Spell  string  `json:"spell"`
}

// A spell's cast as the sim logs it: "[1.50] [Skarr (#1)] Casting {SpellID: 6574} (Cost = ...".
var castLine = regexp.MustCompile(`^\[(-?[\d.]+)\] \[(.+?)(?: \(#\d+\))?\] Casting \{SpellID: (\d+)`)

// The casts of every player in the request's raid, in order, in the first `seconds` of one fight.
func Casts(req *proto.RaidSimRequest, seconds float64) ([]Cast, string) {
	run := googleProto.Clone(req).(*proto.RaidSimRequest)
	if run.SimOptions == nil {
		run.SimOptions = &proto.SimOptions{}
	}
	run.SimOptions.Iterations = 1
	run.SimOptions.Debug = true
	players := map[string]bool{}
	for _, party := range run.GetRaid().GetParties() {
		for _, p := range party.GetPlayers() {
			players[p.GetName()] = true
		}
	}
	result := core.RunRaidSim(run)
	if result.GetError() != nil {
		return nil, result.GetError().GetMessage()
	}
	var out []Cast
	for _, line := range strings.Split(result.GetLogs(), "\n") {
		m := castLine.FindStringSubmatch(line)
		if m == nil || !players[m[2]] {
			continue
		}
		at, _ := strconv.ParseFloat(m[1], 64)
		if at > seconds {
			break
		}
		id, _ := strconv.Atoi(m[3])
		name := strconv.Itoa(id)
		if s := spelldata.Find(int32(id)); s != nil && s.Name != "" {
			name = s.Name
		}
		out = append(out, Cast{At: at, Player: m[2], Spell: name})
	}
	return out, ""
}
