package model

type RewardConfig struct {
	RewardInterval      int
	RewardIntervalPre   int
	ManageFeePercent    int
	ManageFeePercentPre int
	RewardMaturity      int
	DelayInterval       int
	ConfigChangeHeight  int64
}
