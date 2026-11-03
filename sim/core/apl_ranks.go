package core

import (
	"github.com/wowsims/forever/sim/core/proto"
	googleProto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// A spell's family: the spells that are ranks of one another share it, and a rank's level orders them.
type SpellFamily struct {
	Name  string
	Flags ClassFlags
	Level int32
}

var spellFamily func(spellID int32) (SpellFamily, bool)

// The spell rows sit in the spell store, which imports core, so the store registers their reader from
// its init.
func RegisterSpellFamilies(read func(spellID int32) (SpellFamily, bool)) {
	spellFamily = read
}

// Moves each spell the rotation names, and that the character has no spell for, to the highest rank
// of that spell the character has. A rotation written at level 60 names the top ranks, so a character
// below 60 casts the ranks it has learnt. A spell with no rank the character has stays as written.
func (character *Character) rerankRotation(rot *proto.APLRotation) *proto.APLRotation {
	if rot == nil || spellFamily == nil || character.Level >= CharacterLevel {
		return rot
	}
	have := map[int32]bool{}
	best := map[SpellFamily]int32{}
	bestLevel := map[SpellFamily]int32{}
	for _, spell := range character.Spellbook {
		id := spell.ActionID.SpellID
		if id == 0 || have[id] {
			continue
		}
		have[id] = true
		fam, ok := spellFamily(id)
		if !ok {
			continue
		}
		lvl := fam.Level
		fam.Level = 0
		if cur, seen := best[fam]; !seen || lvl > bestLevel[fam] || (lvl == bestLevel[fam] && id > cur) {
			best[fam], bestLevel[fam] = id, lvl
		}
	}
	out := googleProto.Clone(rot).(*proto.APLRotation)
	rerankMessage(out.ProtoReflect(), func(id int32) int32 {
		if have[id] {
			return id
		}
		fam, ok := spellFamily(id)
		if !ok {
			return id
		}
		fam.Level = 0
		if rank, ok := best[fam]; ok {
			return rank
		}
		return id
	})
	return out
}

func rerankMessage(m protoreflect.Message, rank func(int32) int32) {
	if id, ok := m.Interface().(*proto.ActionID); ok {
		if sid, isSpell := id.RawId.(*proto.ActionID_SpellId); isSpell {
			sid.SpellId = rank(sid.SpellId)
		}
		return
	}
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() != protoreflect.MessageKind && fd.Kind() != protoreflect.GroupKind {
			return true
		}
		switch {
		case fd.IsList():
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				rerankMessage(list.Get(i).Message(), rank)
			}
		case fd.IsMap():
			v.Map().Range(func(_ protoreflect.MapKey, mv protoreflect.Value) bool {
				if fd.MapValue().Kind() == protoreflect.MessageKind {
					rerankMessage(mv.Message(), rank)
				}
				return true
			})
		default:
			rerankMessage(v.Message(), rank)
		}
		return true
	})
}
