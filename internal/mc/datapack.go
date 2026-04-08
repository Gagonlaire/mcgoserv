package mc

import "github.com/Gagonlaire/mcgoserv/internal/mcdata"

//field:encode mode=both
type DataPackIdentifier struct {
	Namespace String
	ID        String
	Version   String
}

//field:encode mode=both
type RegistryDataEntry struct {
	ID   String
	Data PrefixedOptional[Byte, *Byte]
}

//field:encode mode=both
type RegistryData struct {
	ID      String
	Entries PrefixedArray[RegistryDataEntry, *RegistryDataEntry]
}

var ServerDataPacks = NewPrefixedArray[DataPackIdentifier, *DataPackIdentifier]([]DataPackIdentifier{
	{
		Namespace: "minecraft",
		ID:        "core",
		Version:   mcdata.GameVersion,
	},
})
