package utils

import "sync"

type ShareDesc struct {
	Username    string
	StartHeight int64
	EndHeight   int64
}

type UserShareIDCache struct {
	cache  map[ShareDesc]uint64
	rwLock sync.RWMutex
}

func NewUserShareIDCache() *UserShareIDCache {
	return &UserShareIDCache{
		cache: make(map[ShareDesc]uint64),
	}
}

func (u *UserShareIDCache) Set(shareDesc ShareDesc, id uint64) {
	u.rwLock.Lock()
	defer u.rwLock.Unlock()
	if len(u.cache) != 0 {
		var tmp ShareDesc
		for s := range u.cache {
			tmp = s
			break
		}
		if tmp.StartHeight != shareDesc.StartHeight {
			u.cache = make(map[ShareDesc]uint64)
		}
	}
	u.cache[shareDesc] = id
}

func (u *UserShareIDCache) Get(shareDesc ShareDesc) (uint64, bool) {
	u.rwLock.RLock()
	defer u.rwLock.RUnlock()
	id, ok := u.cache[shareDesc]
	return id, ok
}

var ShareIDCache = NewUserShareIDCache()
