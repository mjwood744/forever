package mysim

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/hunter"
	"github.com/wowsims/forever/sim/priest"
	"github.com/wowsims/forever/sim/priest/healer"
)

func init() {
	hunter.RegisterHunter()
	priest.RegisterPriest()
	healer.RegisterHealerPriest()
	common.RegisterAllEffects()
}

const bmTalents = "5320001505101251-0053250001-4"

var tankFamilies = []proto.HunterOptions_PetType{
	proto.HunterOptions_Boar, proto.HunterOptions_Turtle, proto.HunterOptions_Bear,
	proto.HunterOptions_Crocilisk, proto.HunterOptions_Wolf, proto.HunterOptions_Cat,
}

func hawke(talents string, pet proto.HunterOptions_PetType) *proto.Player {
	data, _ := os.ReadFile("../../../ui/specs/hunter/dps/apls/turret.apl.json")
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassHunter, Race: proto.Race_RaceWindshaperSkyborne,
		Equipment:   core.GetGearSet("../../../ui/specs/hunter/dps/gear_sets", "p1").GearSet,
		Consumables: &proto.ConsumesSpec{}, Buffs: &proto.IndividualBuffs{},
		TalentsString: talents, DistanceFromTarget: 25, ReactionTimeMs: 100,
		Rotation: core.APLRotationFromJsonString(string(data)),
	}, &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{PetType: pet, PetUptime: 1}}}})
}

// The hunter's (player 0's) first pet, as a unit reference: what Raid.Tanks and a heal target take.
var hunterPetRef = &proto.UnitReference{Type: proto.UnitReference_Pet, Index: 0, Owner: &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0}}

// n level-60 trash mobs, every one on tank slot 0.
func trash(n int, dur float64) *proto.Encounter {
	e := &proto.Encounter{Duration: dur}
	for i := 0; i < n; i++ {
		t := core.FreshDefaultTargetConfig()
		t.Level = 60
		t.MinBaseDamage = 350
		t.CanCrush = false
		t.ParryHaste = false
		e.Targets = append(e.Targets, t)
	}
	return e
}

var scenarios = []struct {
	label string
	mobs  int
	dur   float64
}{{"1 mob 60s", 1, 60}, {"4 mobs 45s", 4, 45}}

func runPetTank(t *testing.T, r *proto.Raid, e *proto.Encounter) (*proto.UnitMetrics, *proto.RaidSimResult) {
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: e, SimOptions: &proto.SimOptions{Iterations: 500, RandomSeed: 3}})
	if res.Error != nil {
		t.Fatal(res.Error.Message)
	}
	return res.RaidMetrics.Parties[0].Players[0], res
}

// The pet tanks alone, with no healer.
func TestPetTank(t *testing.T) {
	fmt.Println("PET|scenario|family|petDTPS|petTPS|petDeath|petDPS|hunterDPS|hunterDTPS")
	for _, sc := range scenarios {
		for _, pet := range tankFamilies {
			r := core.SinglePlayerRaidProto(hawke(bmTalents, pet), &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
			r.Tanks = []*proto.UnitReference{hunterPetRef}
			p, _ := runPetTank(t, r, trash(sc.mobs, sc.dur))
			if len(p.Pets) == 0 {
				t.Fatalf("%v: no pet metrics", pet)
			}
			pm := p.Pets[0]
			fmt.Printf("PET|%s|%v|%.1f|%.1f|%.3f|%.1f|%.1f|%.1f\n", sc.label, pet, pm.Dtps.Avg, pm.Threat.Avg, pm.ChanceOfDeath, pm.Dps.Avg, p.Dps.Avg, p.Dtps.Avg)
		}
	}
}

// A healer priest (player 1) heals the hunter's pet through an APL whose target is the pet, to check
// that heals land on a pet.
const petHealAPL = `{"type":"TypeAPL","priorityList":[
 {"action":{"condition":{"cmp":{"op":"OpLe","lhs":{"currentHealthPercent":{"sourceUnit":PET}},"rhs":{"const":{"val":"50%"}}}},"castSpell":{"spellId":{"spellId":10917},"target":PET}}},
 {"action":{"condition":{"cmp":{"op":"OpLe","lhs":{"currentHealthPercent":{"sourceUnit":PET}},"rhs":{"const":{"val":"80%"}}}},"castSpell":{"spellId":{"spellId":25314},"target":PET}}}
]}`

func healerPriest() *proto.Player {
	data, _ := os.ReadFile("../../../ui/specs/priest/dps/builds/smite_testing.build.json")
	var wrap map[string]json.RawMessage
	_ = json.Unmarshal(data, &wrap)
	p := &proto.Player{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrap["player"], p); err != nil {
		panic(err)
	}
	ref, _ := protojson.Marshal(hunterPetRef)
	apl := string(ref)
	s := petHealAPL
	for i := 0; i+3 <= len(s); i++ {
		if s[i:i+3] == "PET" {
			s = s[:i] + apl + s[i+3:]
			i += len(apl) - 1
		}
	}
	p.TalentsString = "005025021330011531-335001030301"
	p.Consumables = &proto.ConsumesSpec{}
	p.Buffs = &proto.IndividualBuffs{}
	p.Rotation = core.APLRotationFromJsonString(s)
	p.DistanceFromTarget = 20
	p.Spec = &proto.Player_HealerPriest{HealerPriest: &proto.HealerPriest{Options: &proto.HealerPriest_Options{ClassOptions: &proto.PriestOptions{Armor: proto.PriestOptions_InnerFire}}}}
	return p
}

func TestPetTankHealed(t *testing.T) {
	fmt.Println("HEAL|scenario|family|petDTPS|petTPS|petDeath|hunterDPS|priestHPSonPet")
	for _, sc := range scenarios {
		for _, pet := range []proto.HunterOptions_PetType{proto.HunterOptions_Boar, proto.HunterOptions_Turtle, proto.HunterOptions_Bear} {
			party := &proto.Party{Players: []*proto.Player{hawke(bmTalents, pet), healerPriest()}, Buffs: &proto.PartyBuffs{}}
			r := &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}, Tanks: []*proto.UnitReference{hunterPetRef}}
			p, res := runPetTank(t, r, trash(sc.mobs, sc.dur))
			pr := res.RaidMetrics.Parties[0].Players[1]
			// Every heal in the priest's APL targets the pet, so the priest's HPS is healing done to the pet;
			// the pet surviving here and dying in TestPetTank shows it lands.
			pm := p.Pets[0]
			fmt.Printf("HEAL|%s|%v|%.1f|%.1f|%.3f|%.1f|%.1f\n", sc.label, pet, pm.Dtps.Avg, pm.Threat.Avg, pm.ChanceOfDeath, p.Dps.Avg, pr.Hps.Avg)
		}
	}
}
