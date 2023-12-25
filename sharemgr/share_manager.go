package sharemgr

import (
	"sync"
	"time"
)

const (
	IntervalTime = 600
	WindowSize   = 20
)

type Bucket struct {
	sync.Mutex
	startTime int64
	count     int
}

func (b *Bucket) AddCount() {
	b.Lock()
	defer b.Unlock()
	b.count += 1
}

func (b *Bucket) ClearAndAddCount() {
	b.Lock()
	defer b.Unlock()
	b.count = 1
}

func (b *Bucket) GetStartTime() int64 {
	b.Lock()
	defer b.Unlock()
	res := b.startTime
	return res
}

func (b *Bucket) SetStartTime(startTime int64) {
	b.Lock()
	defer b.Unlock()
	b.startTime = startTime
	return
}

func (b *Bucket) ResetStartTime(startTime int64) {
	b.Lock()
	defer b.Unlock()
	b.startTime = startTime
	b.count = 1
}

type ShareManager struct {
	CreateTime time.Time
	BucketNum  int
	Buckets    []*Bucket
}

func NewShareManager() *ShareManager {
	bucketNum := IntervalTime / WindowSize
	buckets := make([]*Bucket, bucketNum)
	for i := 0; i < bucketNum; i++ {
		buckets[i] = &Bucket{}
	}
	return &ShareManager{
		CreateTime: time.Now(),
		BucketNum:  bucketNum,
		Buckets:    buckets,
	}
}

func (m *ShareManager) AddShare() {
	currentTime := time.Now().Unix()
	idx := (currentTime / WindowSize) % int64(m.BucketNum)

	startTime := currentTime - currentTime%WindowSize
	targetBucket := m.Buckets[idx]
	if targetBucket.GetStartTime() == startTime {
		targetBucket.AddCount()
	} else {
		targetBucket.ResetStartTime(startTime)
	}
}

func (m *ShareManager) GetSharePerSecond() float64 {
	totalCount := 0
	currentTime := time.Now().Unix()
	for _, bucket := range m.Buckets {
		if currentTime-bucket.GetStartTime() < IntervalTime {
			totalCount += bucket.count
		}
	}
	if currentTime-m.CreateTime.Unix() < IntervalTime {
		return float64(totalCount) / float64(currentTime-m.CreateTime.Unix())
	}
	return float64(totalCount) / IntervalTime
}

func (m *ShareManager) GetSharePerMinute() float64 {
	return m.GetSharePerSecond() * 60
}
