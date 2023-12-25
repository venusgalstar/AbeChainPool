package constdef

const (
	MinUsernameLength = 6
	MaxUsernameLength = 100
	MinPasswordLength = 6
	MaxPasswordLength = 40
)

const (
	// If BaseDifficulty is changed, BaseHashRate should also be changed.
	BaseDifficulty      uint32 = 0x1e07fffc
	MockMinerDifficulty uint32 = 0x207fffff
	// Unit: kilohash/s
	BaseHashRate                    = 2100
	DefaultDifficultyAdjustInterval = 60
	ShareAdjustThreshold            = 3
	ExpectedSharePerMinute          = 2
)
