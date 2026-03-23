package mc

import "github.com/Gagonlaire/mcgoserv/internal/mcdata"

//go:generate-field-impl
type DataPackIdentifier struct {
	Namespace String
	ID        String
	Version   String
}

//go:generate-field-impl
type RegistryDataEntry struct {
	ID   String
	Data PrefixedOptional[Byte, *Byte]
}

//go:generate-field-impl
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
