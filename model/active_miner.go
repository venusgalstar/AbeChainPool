package model

import (
	"math/big"
	"sync"
	"time"

	"github.com/abesuite/abe-miningpool-server/constdef"
	"github.com/abesuite/abe-miningpool-server/sharemgr"
	"github.com/abesuite/abe-miningpool-server/utils"
)

type ActiveMiner struct {
	UserAgent      string
	Protocol       string
	ConnectionType string
	Address        string
	Username       map[string]time.Time

	ExtraNonce1       string
	ExtraNonce1Uint64 uint64
	ExtraNonce1BitLen int
	ExtraNonce2BitLen int
	Difficulty        int64

	SessionID string

	JobLock sync.Mutex
	// CurrentJob: job_id -> job details
	CurrentJob    map[string]*JobTemplateMiner
	LastNotifyJob *JobTemplate

	ShareCount        int64
	ShareLock         sync.Mutex
	SubmittedShare    map[string]time.Time
	HashRateUpload    int64
	HashRateEstimated float64

	HelloAt      time.Time
	SubscribeAt  time.Time
	AuthorizeAt  time.Time
	LastActiveAt time.Time
	LastNotifyAt time.Time

	ErrorCounts int
	Rejected    int
	Accepted    int

	ShareManager         *sharemgr.ShareManager
	AutoDifficultyAdjust bool

	NotifyStarted           int32
	DifficultyAdjustStarted int32
	MockMiner               bool
}

func (m *ActiveMiner) ClearSubmittedShare() {
	m.ShareLock.Lock()
	defer m.ShareLock.Unlock()

	m.SubmittedShare = make(map[string]time.Time)
}

// AddJob add job to an active miner
func (m *ActiveMiner) AddJob(job *JobTemplate) *JobTemplateMiner {
	if job == nil {
		return nil
	}

	m.JobLock.Lock()
	defer m.JobLock.Unlock()

	if m.Difficulty == 0 {
		m.Difficulty = utils.GetInitialShareDifficulty(m.UserAgent)
		if m.MockMiner {
			m.Difficulty = 1
		}
	}
	targetDifficulty := utils.CompactToBig(constdef.BaseDifficulty)
	targetShare := big.NewInt(0)
	targetShare.Div(targetDifficulty, big.NewInt(m.Difficulty))
	targetShareStr := utils.BigToHashString(targetShare)

	if m.MockMiner {
		targetDifficulty = utils.CompactToBig(constdef.MockMinerDifficulty)
		targetShare = targetDifficulty
		targetShareStr = utils.BigToHashString(targetShare)
	}

	if job.CleanJob || m.CurrentJob == nil {
		m.CurrentJob = make(map[string]*JobTemplateMiner)
	}

	jobMiner := JobTemplateMiner{
		CachedTargetShare:    targetShare,
		CachedTargetShareStr: targetShareStr,
		JobDetails:           job,
	}
	m.CurrentJob[job.JobId] = &jobMiner
	return &jobMiner
}

func (m *ActiveMiner) GetJob() *JobTemplateMiner {
	if len(m.CurrentJob) == 0 {
		return nil
	}
	for _, job := range m.CurrentJob {
		return job
	}
	return nil
}

func (m *ActiveMiner) GetUsername() string {
	if len(m.Username) == 0 {
		return "unknown"
	}
	usernames := make([]string, 0)
	for username, _ := range m.Username {
		usernames = append(usernames, username)
	}
	res := ""
	if len(usernames) > 0 {
		res += usernames[0]
	}
	for i := 1; i < len(usernames); i++ {
		res += " | "
		res += usernames[i]
	}
	if res == "" {
		return "unknown"
	}
	return res
}

func (m *ActiveMiner) SetShareDifficulty(difficulty int64) string {
	if difficulty <= 0 {
		difficulty = 1
	}
	m.Difficulty = difficulty
	targetDifficulty := utils.CompactToBig(constdef.BaseDifficulty)
	targetShare := big.NewInt(0)
	targetShare.Div(targetDifficulty, big.NewInt(m.Difficulty))
	targetShareStr := utils.BigToHashString(targetShare)
	for _, jobTemplateMiner := range m.CurrentJob {
		jobTemplateMiner.CachedTargetShare = targetShare
		jobTemplateMiner.CachedTargetShareStr = targetShareStr
	}
	return targetShareStr
}

func (m *ActiveMiner) EstimateHashRate(cnt float64, sec int) float64 {
	m.HashRateEstimated = constdef.BaseHashRate * float64(m.Difficulty) * cnt / float64(sec)
	return m.HashRateEstimated
}

// CheckDuplicateShare checks if the share has not been submitted before and add it to share map.
func (m *ActiveMiner) CheckDuplicateShare(nonce string, jobID string) bool {
	m.ShareLock.Lock()
	defer m.ShareLock.Unlock()
	shareID := jobID + nonce
	if _, ok := m.SubmittedShare[shareID]; !ok {
		m.SubmittedShare[shareID] = time.Now()
		return true
	}
	return false
}

func (m *ActiveMiner) RejectShare() {
	m.Rejected += 1
}

func (m *ActiveMiner) AcceptShare() {
	m.ShareCount += m.Difficulty
	m.Accepted += 1
}

func (m *ActiveMiner) PrepareWork() {
	if m.Difficulty == 0 {
		m.Difficulty = utils.GetInitialShareDifficulty(m.UserAgent)
	}
}
