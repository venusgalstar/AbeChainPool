package minermgr

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/dal"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/service"
	"github.com/abesuite/abe-miningpool-server/sharemgr"
	"github.com/abesuite/abe-miningpool-server/utils"
	"github.com/abesuite/abe-miningpool-server/wire"

	"gorm.io/gorm"
)

// Constants for the type of a notification message.
const (
	// NTShareDifficultyChanged indicates share difficulty changes.
	NTShareDifficultyChanged NotificationType = iota
	// NTNewJobReady indicates a new job is ready to notify miners.
	NTNewJobReady
	// NTEpochChange indicates epoch changes
	NTEpochChange
)

// notificationTypeStrings is a map of notification types back to their constant
// names for pretty printing.
var notificationTypeStrings = map[NotificationType]string{
	NTShareDifficultyChanged: "NTShareDifficultyChanged",
	NTNewJobReady:            "NTNewJobReady",
	NTEpochChange:            "NTEpochChange",
}

var (
	DefaultExtraNonceBitLen = 64
)

type MinerManager struct {
	ActiveMinerMap        map[chan struct{}]*model.ActiveMiner
	ActiveMinerAddressMap map[string]*model.ActiveMiner
	activeMinerLock       sync.Mutex

	extraNonce1Map    map[string]struct{}
	extraNonce1Lock   sync.Mutex
	extraNonce2BitLen int
	epoch             int64

	JobMap  map[string]*model.JobTemplate
	jobLock sync.Mutex

	cachedTemplate     *model.BlockTemplate
	cachedTemplateLock sync.Mutex

	Db *gorm.DB

	// The notifications field stores a slice of callbacks to be executed on
	// certain events.
	notificationsLock sync.RWMutex
	notifications     []NotificationCallback

	autoDifficultyAdjust bool

	ethash *ethash.Ethash

	mockMiner bool
}

func (mgr *MinerManager) SetEthash(ethash *ethash.Ethash) {
	mgr.ethash = ethash
}

func (mgr *MinerManager) SetCachedTemplate(t *model.BlockTemplate) {
	mgr.cachedTemplateLock.Lock()
	defer mgr.cachedTemplateLock.Unlock()

	mgr.cachedTemplate = t
}

func (mgr *MinerManager) PrepareWork(wsc chan struct{}) error {
	minerInfo, ok := mgr.GetMiner(wsc)
	if !ok {
		log.Errorf("Unable to find certain miner info")
		return pooljson.ErrInternal
	}

	if minerInfo.ExtraNonce1 != "" {
		return nil
	}

	minerInfo.ExtraNonce1BitLen = DefaultExtraNonceBitLen - mgr.extraNonce2BitLen
	minerInfo.ExtraNonce2BitLen = mgr.extraNonce2BitLen
	minerInfo.ExtraNonce1, minerInfo.ExtraNonce1Uint64 = mgr.GenerateExtraNonce(minerInfo.ExtraNonce1BitLen)
	return nil
}

// Subscribe to notifications. Registers a callback to be executed
// when various events take place.
func (mgr *MinerManager) Subscribe(callback NotificationCallback) {
	mgr.notificationsLock.Lock()
	mgr.notifications = append(mgr.notifications, callback)
	mgr.notificationsLock.Unlock()
}

// sendNotification sends a notification with the passed type and data if the
// caller requested notifications by providing a callback function in the call
// to New.
func (mgr *MinerManager) sendNotification(typ NotificationType, data interface{}) {
	// Generate and send the notification.
	n := Notification{Type: typ, Data: data}
	mgr.notificationsLock.RLock()
	for _, callback := range mgr.notifications {
		callback(&n)
	}
	mgr.notificationsLock.RUnlock()
}

func (mgr *MinerManager) AddJob(t *model.BlockTemplate, cleanJob bool) {
	if t == nil {
		log.Errorf("Internal error: nil block template when adding job")
		return
	}

	mgr.jobLock.Lock()
	defer mgr.jobLock.Unlock()

	if len(t.CoinbaseTxn.Data) < int(t.CoinbaseTxn.ExtraNonceOffset*2+t.CoinbaseTxn.ExtraNonceLen*2+1) {
		log.Errorf("Block template error: invalid coinbase data length")
		return
	}

	log.Infof("The coinbase reward of new template is %v Neutrino, Height %v", t.CoinbaseTxn.Fee, t.Height)

	newJob := &model.JobTemplate{}
	newJob.JobId = utils.GenerateJobID()

	// Update coinbase info.
	newJob.Coinbase = t.CoinbaseTxn.Data[:t.CoinbaseTxn.WitnessOffset*2]
	newJob.CoinbaseWithWitness = t.CoinbaseTxn.Data

	// Update extra nonce in coinbase
	extraNonceInCoinbase := utils.RandHexString(int(t.CoinbaseTxn.ExtraNonceLen * 2))
	newJob.Coinbase = newJob.Coinbase[:t.CoinbaseTxn.ExtraNonceOffset*2] + extraNonceInCoinbase + newJob.Coinbase[t.CoinbaseTxn.ExtraNonceOffset*2+t.CoinbaseTxn.ExtraNonceLen*2:]
	newJob.CoinbaseWithWitness = newJob.CoinbaseWithWitness[:t.CoinbaseTxn.ExtraNonceOffset*2] + extraNonceInCoinbase + newJob.CoinbaseWithWitness[t.CoinbaseTxn.ExtraNonceOffset*2+t.CoinbaseTxn.ExtraNonceLen*2:]

	// Update previous hash info
	newJob.PreviousHash = t.PreviousHash
	previousBlock, err := utils.NewHashFromStr(newJob.PreviousHash)
	if err != nil {
		log.Errorf("Internal error: incorrect previous hash in block template, %v", err)
		return
	}
	newJob.PreviousHashH = *previousBlock

	// Update tx hashes
	txHashes := make([]utils.Hash, 0)
	coinbaseBytes, err := hex.DecodeString(newJob.Coinbase)
	if err != nil {
		log.Errorf("Abec submits a block template with invalid coinbase %v ", newJob.Coinbase)
		return
	}
	coinbaseHash := utils.DoubleHashH(coinbaseBytes)
	txHashes = append(txHashes, coinbaseHash)
	for _, tx := range t.Transactions {
		tmp, err := utils.NewHashFromStr(tx.TxHash)
		if err != nil {
			log.Errorf("Internal error: incorrect transaction hash in block template, %v", err)
			return
		}
		txHashes = append(txHashes, *tmp)
	}
	newJob.TxHashes = txHashes

	// Update merkle root.
	witnessHashes := make([]utils.Hash, 0)
	cbWitnessHash, err := utils.NewHashFromStr(t.CoinbaseTxn.WitnessHash)
	if err != nil {
		log.Errorf("Abec submits a block template with invalid coinbase witness hash %v ", t.CoinbaseTxn.WitnessHash)
		return
	}
	witnessHashes = append(witnessHashes, *cbWitnessHash)
	for _, tx := range t.Transactions {
		witnessHash, err := utils.NewHashFromStr(tx.WitnessHash)
		if err != nil {
			log.Errorf("Abec submits a block template with invalid tx witness hash %v ", tx.WitnessHash)
			return
		}
		witnessHashes = append(witnessHashes, *witnessHash)
	}
	merkleRoot, err := utils.BuildMerkleTreeStore(txHashes, witnessHashes)

	if err != nil {
		log.Errorf("Error occurs when extract merkle root: %v", err)
		return
	}
	newJob.MerkleRoot = merkleRoot.String()
	newJob.MerkleRootH = merkleRoot

	// Update version, time and bits
	newJob.Version = t.Version
	newJob.CurrTime = time.Now()
	newJob.TargetDifficulty = t.Bits
	bits, err := strconv.ParseInt(t.Bits, 16, 64)
	if err != nil {
		log.Errorf("Error occurs when convert bits string to int64")
		return
	}
	newJob.Bits = uint32(bits)
	targetDifficulty := utils.CompactToBig(uint32(bits))
	newJob.CacheTargetDifficulty = targetDifficulty

	newJob.Height = t.Height
	newJob.Reward = t.CoinbaseTxn.Fee
	newJob.CleanJob = cleanJob

	blockHeader := wire.BlockHeader{
		Version:    newJob.Version,
		PrevBlock:  newJob.PreviousHashH,
		MerkleRoot: newJob.MerkleRootH,
		Timestamp:  newJob.CurrTime,
		Bits:       newJob.Bits,
		Height:     int32(newJob.Height),
	}
	newJob.ContentHashH = blockHeader.ContentHash()
	newJob.ContentHash = hex.EncodeToString(newJob.ContentHashH[:])

	if cleanJob {
		mgr.JobMap = make(map[string]*model.JobTemplate)
	}
	mgr.JobMap[newJob.JobId] = newJob
	mgr.SetCachedTemplate(t)
	newEpoch := int64(ethash.CalculateEpoch(int32(newJob.Height)))
	if mgr.epoch >= 0 && newEpoch != mgr.epoch {
		mgr.sendNotification(NTEpochChange, newEpoch)
	}
	mgr.epoch = newEpoch

	mgr.sendNotification(NTNewJobReady, newJob)
	mgr.RefreshNotifyTime()
}

func (mgr *MinerManager) RefreshNotifyTime() {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()

	newTime := time.Now()
	for _, minerInfo := range mgr.ActiveMinerMap {
		if minerInfo.AuthorizeAt.IsZero() {
			continue
		}
		minerInfo.LastNotifyAt = newTime
	}
}

func (mgr *MinerManager) SwitchJob(miner chan struct{}, job *model.JobTemplate) {
	minerInfo, ok := mgr.GetMiner(miner)
	if !ok {
		log.Errorf("Internal error: unable to find certain miner when switching job.")
		return
	}

	currJob := minerInfo.CurrentJob
	_, ok = currJob[job.JobId]
	if ok {
		return
	}

	minerInfo.AddJob(job)
}

func (mgr *MinerManager) RejectBlock(blockNotification *model.BlockNotification) error {
	db := mgr.Db
	userService := service.GetUserService()

	err := userService.RejectBlock(context.Background(), db, blockNotification)
	return err
}

func (mgr *MinerManager) ConnectBlock(blockNotification *model.BlockNotification) error {
	db := mgr.Db
	userService := service.GetUserService()

	err := userService.ConnectBlock(context.Background(), db, blockNotification.BlockHash.String())
	return err
}

func (mgr *MinerManager) DisconnectBlock(blockNotification *model.BlockNotification) error {
	db := mgr.Db
	userService := service.GetUserService()

	err := userService.DisconnectBlock(context.Background(), db, blockNotification.BlockHash.String())
	return err
}

// ClearSubmittedShare clear all the submitted share recorded before.
func (mgr *MinerManager) ClearSubmittedShare() {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()
	for _, m := range mgr.ActiveMinerMap {
		m.ClearSubmittedShare()
	}
}

// CheckShareValidity checks if the share meets the share difficulty and block difficulty.
// If is valid share or block, return the details.
func (mgr *MinerManager) CheckShareValidity(nonce string, job *model.JobTemplateMiner, minerInfo *model.ActiveMiner,
	username string) (bool, bool, *model.ShareInfo, error) {

	nonceUint64, err := strconv.ParseUint(nonce, 16, 64)
	if err != nil {
		return false, false, nil, pooljson.ErrInvalidShare
	}
	if (nonceUint64 >> (minerInfo.ExtraNonce2BitLen)) != minerInfo.ExtraNonce1Uint64 {
		return false, false, nil, pooljson.ErrInvalidShare
	}
	extNonce := nonce

	n, err := strconv.ParseUint(extNonce, 16, 64)
	if err != nil {
		log.Debugf("Error occurs when decoding nonce %v: %v", extNonce, err)
		return false, false, nil, pooljson.ErrInvalidRequestParams
	}

	header := &wire.BlockHeader{
		Version:    job.JobDetails.Version,
		PrevBlock:  job.JobDetails.PreviousHashH,
		MerkleRoot: job.JobDetails.MerkleRootH,
		Timestamp:  job.JobDetails.CurrTime,
		Height:     int32(job.JobDetails.Height),
		Bits:       job.JobDetails.Bits,
		NonceExt:   n,
	}

	sealHash, mixDigest, err := mgr.ethash.CalculateSeal(header)
	if err != nil {
		log.Errorf("Internal error: calculate seal hash fail: %v", err)
		return false, false, nil, pooljson.ErrInternal
	}
	header.MixDigest = utils.Hash{}
	copy(header.MixDigest[:], mixDigest)
	shareHash := header.BlockHash()

	isValidBlock := false
	isValidShare := false
	shareInfo := &model.ShareInfo{
		Username:    username,
		Height:      job.JobDetails.Height,
		ShareCount:  minerInfo.Difficulty,
		ContentHash: job.JobDetails.ContentHash,
		ShareHash:   shareHash.String(),
		SealHash:    sealHash.String(),
		Reward:      job.JobDetails.Reward,
		Nonce:       extNonce,
	}
	if utils.HashToBig(&sealHash).Cmp(job.CachedTargetShare) <= 0 {
		isValidShare = true
		if utils.HashToBig(&sealHash).Cmp(job.JobDetails.CacheTargetDifficulty) <= 0 {
			isValidBlock = true
			coinbaseByte, err := hex.DecodeString(job.JobDetails.CoinbaseWithWitness)
			if err != nil {
				log.Errorf("Unable to decode coinbase: %v", coinbaseByte)
				return false, false, nil, pooljson.ErrInternal
			}
			newReader := bytes.NewReader(coinbaseByte)
			cb := &wire.MsgTxAbe{}
			err = cb.Deserialize(newReader)
			if err != nil {
				log.Errorf("Unable to decode coinbase: %v", coinbaseByte)
				return false, false, nil, pooljson.ErrInternal
			}
			submitCandidate := &wire.MsgSimplifiedBlock{
				Header:       *header,
				Coinbase:     cb,
				Transactions: job.JobDetails.TxHashes,
			}
			shareInfo.CandidateBlock = submitCandidate
		}
	}

	return isValidShare, isValidBlock, shareInfo, nil
}

func SetupMinerManger(autoDifficultyAdjust bool, extraNonce2BitLen int, mockMiner bool) *MinerManager {
	if mockMiner {
		autoDifficultyAdjust = false
	}
	minerMgr := &MinerManager{
		extraNonce2BitLen:     extraNonce2BitLen,
		ActiveMinerMap:        make(map[chan struct{}]*model.ActiveMiner),
		ActiveMinerAddressMap: make(map[string]*model.ActiveMiner),
		extraNonce1Map:        make(map[string]struct{}),
		epoch:                 -1,
		JobMap:                make(map[string]*model.JobTemplate),
		Db:                    dal.GlobalDBClient,
		autoDifficultyAdjust:  autoDifficultyAdjust,
		mockMiner:             mockMiner,
	}

	return minerMgr
}

func (mgr *MinerManager) GenerateExtraNonce(length int) (string, uint64) {
	mgr.extraNonce1Lock.Lock()
	defer mgr.extraNonce1Lock.Unlock()

	var extraNonce1 string
	var extraNonce1Uint64 uint64
	for {
		extraNonce1, extraNonce1Uint64 = utils.GenerateExtraNonce1(length)
		if _, ok := mgr.extraNonce1Map[extraNonce1]; !ok {
			mgr.extraNonce1Map[extraNonce1] = struct{}{}
			break
		}
	}

	return extraNonce1, extraNonce1Uint64
}

func (mgr *MinerManager) GenerateNewSession(wsc chan struct{}) (string, error) {
	miner, ok := mgr.GetMiner(wsc)
	if !ok {
		return "", pooljson.ErrHelloNotReceived
	}

	miner.SessionID = "s-" + utils.RandStr(8)
	miner.SubscribeAt = time.Now()
	return miner.SessionID, nil
}

func (mgr *MinerManager) GetMiner(wsc chan struct{}) (*model.ActiveMiner, bool) {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()
	miner, ok := mgr.ActiveMinerMap[wsc]
	return miner, ok
}

func (mgr *MinerManager) SetHashRate(wsc chan struct{}, hashRate int64) error {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()
	miner, ok := mgr.ActiveMinerMap[wsc]
	if !ok || miner == nil {
		return errors.New("miner not found")
	}
	miner.HashRateUpload = hashRate
	return nil
}

// AddNewMiner add a new miner into ActiveMinerMap.
func (mgr *MinerManager) AddNewMiner(wsc chan struct{}, remoteAddress string, userAgent string, protocol string, connectionType string) (*model.ActiveMiner, error) {

	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()

	num := len(mgr.ActiveMinerMap)
	var maxMiner uint64 = (1 << (DefaultExtraNonceBitLen - mgr.extraNonce2BitLen)) - 1
	threshold := uint64(float64(maxMiner) * 0.98)
	if uint64(num) >= threshold {
		log.Warnf("Miner exceeds limit %v, reject", threshold)
		return nil, pooljson.ErrExceedMinerLimit
	}

	newMiner := &model.ActiveMiner{
		UserAgent:            userAgent,
		Protocol:             protocol,
		ConnectionType:       connectionType,
		Address:              remoteAddress,
		Username:             make(map[string]time.Time),
		CurrentJob:           make(map[string]*model.JobTemplateMiner),
		SubmittedShare:       make(map[string]time.Time),
		HelloAt:              time.Now(),
		LastActiveAt:         time.Now(),
		ShareManager:         sharemgr.NewShareManager(),
		AutoDifficultyAdjust: mgr.autoDifficultyAdjust,
		ErrorCounts:          0,
		MockMiner:            mgr.mockMiner,
	}

	if mgr.ActiveMinerMap == nil {
		mgr.ActiveMinerMap = make(map[chan struct{}]*model.ActiveMiner)
	}
	if mgr.ActiveMinerAddressMap == nil {
		mgr.ActiveMinerAddressMap = make(map[string]*model.ActiveMiner)
	}
	mgr.ActiveMinerMap[wsc] = newMiner
	mgr.ActiveMinerAddressMap[remoteAddress] = newMiner
	return newMiner, nil
}

// DeleteActiveMiner delete a certain miner from ActiveMinerMap.
// Usually used when a miner disconnected.
func (mgr *MinerManager) DeleteActiveMiner(wsc chan struct{}) {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()

	minerInfo, ok := mgr.ActiveMinerMap[wsc]
	if !ok {
		return
	}
	extraNonce1 := minerInfo.ExtraNonce1
	remoteAddr := minerInfo.Address
	mgr.extraNonce1Lock.Lock()
	defer mgr.extraNonce1Lock.Unlock()

	delete(mgr.extraNonce1Map, extraNonce1)
	delete(mgr.ActiveMinerMap, wsc)
	delete(mgr.ActiveMinerAddressMap, remoteAddr)
}

// GetMinerNum return the current number of miners.
func (mgr *MinerManager) GetMinerNum() int {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()
	res := len(mgr.ActiveMinerMap)
	return res
}

// GetMiners return all the addresses of connected miners.
func (mgr *MinerManager) GetMiners() []string {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()
	res := make([]string, 0)
	for _, miner := range mgr.ActiveMinerMap {
		res = append(res, miner.Address)
	}
	return res
}

// GetActiveMiner return the detail info of active miner.
func (mgr *MinerManager) GetActiveMiner(reomteAddress string) *model.ActiveMiner {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()
	res, ok := mgr.ActiveMinerAddressMap[reomteAddress]
	if !ok {
		return nil
	}
	return res
}

// GetAllMiners return the detail info of all active miners.
func (mgr *MinerManager) GetAllMiners() []*model.ActiveMiner {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()
	res := make([]*model.ActiveMiner, 0)
	for _, miner := range mgr.ActiveMinerMap {
		res = append(res, miner)
	}
	return res
}

// GetHashRate return the estimated hash rate of mining pool.
func (mgr *MinerManager) GetHashRate() int64 {
	mgr.activeMinerLock.Lock()
	defer mgr.activeMinerLock.Unlock()
	res := 0.0
	for _, miner := range mgr.ActiveMinerMap {
		res += miner.HashRateEstimated
	}
	return int64(res)
}

// HandleChainClientNotification handles notifications from chain client.  It does
// things such as notify template changed and so on.
func (mgr *MinerManager) HandleChainClientNotification(notification *chainclient.Notification) {
	switch notification.Type {

	case chainclient.NTBlockTemplateChanged:
		log.Debug("Miner manager receives a new block template from chain client, updating...")
		blockTemplate, ok := notification.Data.(*model.BlockTemplate)
		if !ok {
			log.Errorf("Miner manager accepted notification is not a block template.")
			break
		}
		if blockTemplate == nil {
			log.Errorf("Received block template is nil")
			break
		}
		mgr.AddJob(blockTemplate, true)

	case chainclient.NTBlockAccepted:
		log.Debug("Miner manager receives a block accepted notification from chain client.")
		_, ok := notification.Data.(*model.BlockNotification)
		if !ok {
			log.Errorf("Miner manager accepted notification is not a block notification.")
			break
		}
		mgr.ClearSubmittedShare()

	case chainclient.NTBlockRejected:
		log.Debug("Miner manager receives a block rejected notification from chain client.")
		blockNotification, ok := notification.Data.(*model.BlockNotification)
		if !ok {
			log.Errorf("Miner manager accepted notification is not a block notification.")
			break
		}
		err := mgr.RejectBlock(blockNotification)
		if err != nil {
			log.Errorf("Reject block %v fail: %v", blockNotification.BlockHash.String(), err)
		}

	case chainclient.NTBlockConnected:
		log.Debug("Miner manager receives a block connected notification from chain client.")
		blockNotification, ok := notification.Data.(*model.BlockNotification)
		if !ok {
			log.Errorf("Miner manager accepted notification is not a block notification.")
			break
		}
		err := mgr.ConnectBlock(blockNotification)
		if err != nil {
			log.Errorf("Update status of block %v to connected fail: %v", blockNotification.BlockHash.String(), err)
		}

	case chainclient.NTBlockDisconnected:
		log.Debug("Miner manager receives a block disconnected notification from chain client.")
		blockNotification, ok := notification.Data.(*model.BlockNotification)
		if !ok {
			log.Errorf("Miner manager accepted notification is not a block notification.")
			break
		}
		err := mgr.DisconnectBlock(blockNotification)
		if err != nil {
			log.Errorf("Update status of block %v to disconnected fail: %v", blockNotification.BlockHash.String(), err)
		}
	}
}

func (mgr *MinerManager) GetExtraNonce2BitLen() int {
	return mgr.extraNonce2BitLen
}

// GetJob return a random job template from jobMap
func (mgr *MinerManager) GetJob() *model.JobTemplate {
	if len(mgr.JobMap) == 0 {
		return nil
	}

	mgr.jobLock.Lock()
	defer mgr.jobLock.Unlock()
	var res *model.JobTemplate
	for _, entry := range mgr.JobMap {
		res = entry
		break
	}
	return res
}
