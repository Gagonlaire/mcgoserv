package container

import (
	"io"
	"reflect"

	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type Slot struct {
	Components *map[int32]any
	RemoveList *[]int32
	Count      int32
	ItemID     int32
}

// StacksWith reports whether s and other are mergeable (same ItemID + equal components).
// todo: RemoveList equality uses reflect.DeepEqual which is order-sensitive, refactor to a set
func (s Slot) StacksWith(other Slot) bool {
	if s.Count <= 0 || other.Count <= 0 {
		return false
	}
	if s.ItemID != other.ItemID {
		return false
	}
	if s.Components == nil && other.Components == nil && s.RemoveList == nil && other.RemoveList == nil {
		return true
	}
	return reflect.DeepEqual(s.Components, other.Components) && reflect.DeepEqual(s.RemoveList, other.RemoveList)
}

func (s *Slot) ReadFrom(r io.Reader) (n int64, err error) {
	var count, itemID, componentToAdd, componentToRemove proto.VarInt
	nn, err := count.ReadFrom(r)
	n += nn
	if err != nil {
		return nn, err
	}
	s.Count = int32(count)
	if count <= 0 {
		return
	}
	nn, err = itemID.ReadFrom(r)
	n += nn
	if err != nil {
		return n, err
	}
	s.ItemID = int32(itemID)
	nn, err = componentToAdd.ReadFrom(r)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = componentToRemove.ReadFrom(r)
	n += nn
	if err != nil {
		return n, err
	}
	// todo: component to add/remove should not be higher than 0 for now
	return n, nil
}

func (s Slot) WriteTo(w io.Writer) (n int64, err error) {
	nn, err := proto.VarInt(s.Count).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	if s.Count <= 0 {
		return n, nil
	}
	nn, err = proto.VarInt(s.ItemID).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = proto.VarInt(0).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = proto.VarInt(0).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	return n, nil
}
