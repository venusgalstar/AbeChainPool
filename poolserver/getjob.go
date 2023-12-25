package poolserver

import (
	"strconv"

	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/pooljson"
)

// handleGetJob implements the mining.get_job command.
func handleGetJob(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	_, ok := icmd.(*pooljson.GetJobCmd)
	if !ok {
		log.Debugf("Client %v sends an invalid mining.get_job command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInternal
	}

	minerMgr := wsc.GetMinerManager()
	minerInfo, ok := minerMgr.GetMiner(wsc.QuitChan())
	if !ok {
		log.Debugf("Client %v sends a mining.get_job command but not subscribed.", wsc.RemoteAddr())
		return nil, pooljson.ErrUnsubscribed
	}
	if len(minerInfo.Username) == 0 {
		log.Debugf("Client %v get job without authorizing.", wsc.RemoteAddr())
		return nil, pooljson.ErrUnauthorized
	}

	log.Debugf("Receive mining.get_job from %v", wsc.RemoteAddr())
	job := minerInfo.GetJob()
	if job != nil {
		heightSent := strconv.FormatInt(job.JobDetails.Height, 16)
		epoch := ethash.CalculateEpoch(int32(job.JobDetails.Height))
		epochSent := strconv.FormatInt(int64(epoch), 16)
		return &pooljson.GetJobResult{
			JobId:       job.JobDetails.JobId,
			Height:      heightSent,
			Epoch:       epochSent,
			ContentHash: job.JobDetails.ContentHash,
			Target:      job.CachedTargetShareStr,
			ExtraNonce1: minerInfo.ExtraNonce1,
		}, nil
	}

	jobNew := minerMgr.GetJob()
	if jobNew != nil {
		jobMiner := minerInfo.AddJob(jobNew)
		heightSent := strconv.FormatInt(jobNew.Height, 16)
		epoch := ethash.CalculateEpoch(int32(jobNew.Height))
		epochSent := strconv.FormatInt(int64(epoch), 16)
		return &pooljson.GetJobResult{
			JobId:       jobNew.JobId,
			Height:      heightSent,
			Epoch:       epochSent,
			ContentHash: jobNew.ContentHash,
			Target:      jobMiner.CachedTargetShareStr,
			ExtraNonce1: minerInfo.ExtraNonce1,
		}, nil
	}

	log.Errorf("Internal error: unable to find job to response")
	return nil, pooljson.ErrInternal
}
