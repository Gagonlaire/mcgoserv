package mc

import (
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

//field:encode mode=both
type DataPackIdentifier struct {
	Namespace proto.String
	ID        proto.String
	Version   proto.String
}

//field:encode mode=both
type RegistryDataEntry struct {
	ID   proto.String
	Data proto.PrefixedOptional[proto.Byte, *proto.Byte]
}

//field:encode mode=both
type RegistryData struct {
	ID      proto.String
	Entries proto.PrefixedArray[RegistryDataEntry, *RegistryDataEntry]
}

//field:encode mode=both
type TagEntry struct {
	Name    proto.String
	Entries proto.PrefixedArray[proto.VarInt, *proto.VarInt]
}

//field:encode mode=both
type RegistryTags struct {
	ID   proto.String
	Tags proto.PrefixedArray[TagEntry, *TagEntry]
}

var ServerDataPacks = proto.NewPrefixedArray[DataPackIdentifier, *DataPackIdentifier]([]DataPackIdentifier{
	{
		Namespace: "minecraft",
		ID:        "core",
		Version:   mcdata.GameVersion,
	},
})
