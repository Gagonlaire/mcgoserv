package mc

import "io"

type Slot struct {
	Components *map[int32]any
	RemoveList *[]int32
	Count      int32
	ItemID     int32
}

func (s *Slot) ReadFrom(r io.Reader) (n int64, err error) {
	var count, itemID, componentToAdd, componentToRemove VarInt
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
	nn, err := VarInt(s.Count).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	if s.Count <= 0 {
		return n, nil
	}
	nn, err = VarInt(s.ItemID).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = VarInt(0).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = VarInt(0).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	return n, nil
}
