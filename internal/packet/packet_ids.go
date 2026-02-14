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
	LoginClientboundDisconnect        = 0x00
	LoginServerboundLoginStart        = 0x00
	LoginServerboundLoginAcknowledged = 0x03
)

const (
	ConfigurationClientboundFinishConfiguration            = 0x03
	ConfigurationClientboundKeepAlive                      = 0x04
	ConfigurationClientboundDisconnect                     = 0x02
	ConfigurationClientboundRegistryData                   = 0x07
	ConfigurationClientboundKnownPacks                     = 0x0E
	ConfigurationServerboundAcknowledgeFinishConfiguration = 0x03
	ConfigurationServerboundKeepAlive                      = 0x04
	ConfigurationServerboundKnownPacks                     = 0x07
)

const (
	PlayClientboundSetChunkCacheCenter        = 0x5C
	PlayClientboundSpawnEntity                = 0x01
	PlayClientboundGameEvent                  = 0x26
	PlayClientboundUpdateEntityPosition       = 0x33
	PlayClientboundUpdateEntityPositionAndRot = 0x34
	PlayClientboundUpdateEntityRotation       = 0x36
	PlayClientboundKeepAlive                  = 0x2B
	PlayClientboundDisconnect                 = 0x20
	PlayClientboundChunkDataAndUpdateLight    = 0x2C
	PlayClientboundLogin                      = 0x30
	PlayClientboundPlayerInfoRemove           = 0x43
	PlayClientboundPlayerInfoUpdate           = 0x44
	PlayClientboundSynchronizePlayerPosition  = 0x46
	PlayClientboundRotateHead                 = 0x51
	PlayClientboundSetEntityData              = 0x61
	PlayClientboundSetTime                    = 0x6F
	PlayClientboundSystemChat                 = 0x77
	PlayClientboundTeleportEntity             = 0x23
	PlayClientboundRemoveEntities             = 0x4B
	PlayClientboundAnimate                    = 0x02
	PlayClientboundAcknowledgeBlockChange     = 0x04
	PlayClientboundBlockUpdate                = 0x08
	PlayServerboundPlayerAction               = 0x28
	PlayServerboundConfirmTeleportation       = 0x00
	PlayServerboundClientTickEnd              = 0x0C
	PlayServerboundKeepAlive                  = 0x1B
	PlayServerboundMovePlayerPos              = 0x1D
	PlayServerboundMovePlayerPosRot           = 0x1E
	PlayServerboundMovePlayerRot              = 0x1F
	PlayServerboundMovePlayerStatusOnly       = 0x20
	PlayServerboundPlayerCommand              = 0x29
	PlayServerboundPlayerInput                = 0x2A
	PlayServerboundPlayerLoaded               = 0x2B
	PlayServerboundSwingArm                   = 0x3C
	PlayServerboundChat                       = 0x08
)
