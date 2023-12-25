package model

type DifficultyNotification struct {
	Client     chan struct{}
	Difficulty int64
}
