package nbtpath

import (
	"bytes"

	"github.com/Tnze/go-mc/nbt"
)

func SNBTToValue(snbt nbt.StringifiedMessage) (any, error) {
	var buf bytes.Buffer
	tagType := snbt.TagType()
	if tagType == nbt.TagEnd {
		return nil, &nbt.SyntaxError{Message: "invalid SNBT"}
	}

	buf.WriteByte(tagType)
	buf.WriteByte(0)
	buf.WriteByte(0)
	if err := snbt.MarshalNBT(&buf); err != nil {
		return nil, err
	}

	var value any
	if err := nbt.Unmarshal(buf.Bytes(), &value); err != nil {
		return nil, err
	}
	return value, nil
}

// MarshalToValue convert a Go value (ex: an entity struct) into a typed tree.
func MarshalToValue(v any) (any, error) {
	bin, err := nbt.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := nbt.Unmarshal(bin, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

// WriteBack serializes m via NBT then unmarshals it back into dst
func WriteBack(dst any, m any) error {
	bin, err := nbt.Marshal(m)
	if err != nil {
		return err
	}
	return nbt.Unmarshal(bin, dst)
}
