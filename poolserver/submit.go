package poolserver

import (
	"context"
	"time"

	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/service"
	"github.com/abesuite/abe-miningpool-server/utils"
)

// handleSubmit implements the mining.submit command.
func handleSubmit(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.SubmitCmd)
	if !ok {
		log.Debugf("Client %v sends an invalid mining.submit command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	chainClient := wsc.GetChainClient()
	if chainClient == nil {
		log.Warn("Chain client is nil, maybe Abec is not connected.")
		return nil, pooljson.ErrInternal
	}

	minerMgr := wsc.GetMinerManager()
	minerInfo, ok := minerMgr.GetMiner(wsc.QuitChan())
	if !ok {
		log.Debugf("Client %v submits a share without subscription.", wsc.RemoteAddr())
		return nil, pooljson.ErrUnsubscribed
	}

	_, ok = minerInfo.Username[cmd.WorkerID]
	if !ok {
		log.Debugf("Client %v submits a share with wrong username %v.", wsc.RemoteAddr(), cmd.WorkerID)
		return nil, pooljson.ErrUnauthorized
	}

	if utils.IsBlank(cmd.JobID) || utils.IsBlank(cmd.Nonce) {
		log.Debugf("Client %v submits a share with invalid blank params.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	job, ok := minerInfo.CurrentJob[cmd.JobID]
	if !ok {
		log.Debugf("Client %v submits a share with job %v not found.", wsc.RemoteAddr(), cmd.JobID)
		return nil, pooljson.ErrJobNotFound
	}

	// Check duplicate share and add it to share map.
	notSubmitted := minerInfo.CheckDuplicateShare(cmd.Nonce, cmd.JobID)
	if !notSubmitted {
		minerInfo.RejectShare()
		log.Debugf("Client %v submits a duplicate share.", wsc.RemoteAddr())
		return nil, pooljson.ErrDuplicateShare
	}

	// Check share validity.
	isValidShare, isValidBlock, shareInfo, err := minerMgr.CheckShareValidity(cmd.Nonce, job, minerInfo, cmd.WorkerID)
	if err != nil {
		return nil, err
	}

	db := wsc.GetDB()
	userService := service.GetUserService()
	// If valid, update the database.
	if isValidShare {
		log.Debugf("Share %v (Seal hash: %v) submitted by client %v (%v) is valid.", shareInfo.ShareHash, shareInfo.SealHash,
			wsc.RemoteAddr(), cmd.WorkerID)
		minerInfo.ShareManager.AddShare()

		// Record share info
		var realRewardInterval int
		if shareInfo.Height >= wsc.GetRewardIntervalChangeHeight() {
			realRewardInterval = wsc.GetRewardInterval()
		} else {
			realRewardInterval = wsc.GetRewardIntervalPre()
		}
		err := userService.AddShareByUsername(context.Background(), db, shareInfo, realRewardInterval, wsc.DetailedShareInfo())
		if err != nil {
			return nil, pooljson.ErrInternal
		}

		minerInfo.AcceptShare()
	} else {
		log.Debugf("Share submitted by client %v (%v) is invalid.", wsc.RemoteAddr(), cmd.WorkerID)
		return nil, pooljson.ErrInvalidShare
	}
	minerInfo.LastActiveAt = time.Now()

	// If is a valid block, submit to abec and update the database.
	if isValidBlock {
		if shareInfo.CandidateBlock == nil {
			log.Errorf("Internal error: nil candidate block.")
			return nil, pooljson.ErrInternal
		}
		log.Infof("Share %v (Seal hash: %v) submitted by client %v (%v) is a valid block, submitting to abec...",
			shareInfo.ShareHash, shareInfo.SealHash, wsc.RemoteAddr(), cmd.WorkerID)
		err := userService.AddMinedBlock(context.Background(), db, shareInfo)
		if err != nil {
			log.Errorf("Internal error: add mined block fail, %v", err)
			return nil, pooljson.ErrInternal
		}
		submittedBlockHash, err := utils.NewHashFromStr(shareInfo.ShareHash)
		if err != nil {
			log.Errorf("Internal error: utils.NewHashFromStr(shareInfo.ShareHash) fail, %v", err)
			return nil, pooljson.ErrInternal
		}
		submitContent := model.BlockSubmitted{
			BlockHash:       *submittedBlockHash,
			SimplifiedBlock: shareInfo.CandidateBlock,
		}
		chainClient.SubmitBlock(&submitContent)
	}

	var detail string
	if isValidBlock {
		detail = "Valid block"
	} else {
		detail = "Valid share"
	}
	res := pooljson.SubmitResult{
		Detail: detail,
	}
	return &res, nil
}
