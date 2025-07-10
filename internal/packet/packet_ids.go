package packet

const (
	HandshakeServerboundHandshake = 0x00
)

const (
	StatusClientboundStatusResponse = 0x00
	StatusClientboundPongResponse   = 0x01
	StatusServerboundStatusRequest  = 0x00
	StatusServerboundPing           = 0x01
)

const (
	LoginClientboundLoginSuccess      = 0x02
	LoginServerboundLoginStart        = 0x00
	LoginServerboundLoginAcknowledged = 0x03
)

const (
	ConfigurationClientboundFinishConfiguration            = 0x03
	ConfigurationClientboundRegistryData                   = 0x07
	ConfigurationClientboundKnownPacks                     = 0x0E
	ConfigurationServerboundAcknowledgeFinishConfiguration = 0x03
	ConfigurationServerboundKeepAlive                      = 0x04
	ConfigurationServerboundKnownPacks                     = 0x07
)

const (
	PlayClientboundGameEvent                 = 0x22
	PlayClientboundChunkDataAndUpdateLight   = 0x27
	PlayClientboundLogin                     = 0x2B
	PlayClientboundSynchronizePlayerPosition = 0x41
	PlayServerboundConfirmTeleportation      = 0x00
	PlayServerboundKeepAlive                 = 0x1B
)
