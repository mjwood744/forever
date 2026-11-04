package trios

import (
	"reflect"
	"testing"
)

// A Warrior gets Skarr's keys for the level: the keys the sims press in the order they check them, the
// mashed key last of those, then the keys kept for when something happens.
func TestKeysForAWarriorAreSkarrs(t *testing.T) {
	k, ok := KeysFor("WARRIOR", 32)
	if !ok {
		t.Fatal("no keys for a Warrior")
	}
	if k.From != "Skarr" {
		t.Errorf("keys from %s, want Skarr", k.From)
	}
	var order []string
	for _, u := range k.Keys {
		order = append(order, u.Key)
	}
	if want := []string{"Shift+2", "2", "1", "E", "R", "T"}; !reflect.DeepEqual(order, want) {
		t.Errorf("keys %v, want %v", order, want)
	}
	main := k.Keys[2]
	if !main.Main || main.When != "" || len(main.Spells) == 0 {
		t.Errorf("key 1 is %+v, want the mashed key with its spells", main)
	}
	if got := k.Keys[0].Spells; !reflect.DeepEqual(got, []string{"Shield Wall"}) {
		t.Errorf("Shift+2 casts %v, want Shield Wall", got)
	}
}
