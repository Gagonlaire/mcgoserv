package mc

const (
	ProtocolVersion = 770
	GameVersion     = "1.21.5"
)

var ServerDataPacks = []DataPackIdentifier{
	{
		Namespace: String("minecraft"),
		ID:        String("core"),
		Version:   String(GameVersion),
	},
}
