package poolserver

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/abesuite/abe-miningpool-server/constdef"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/service"
	"github.com/abesuite/abe-miningpool-server/utils"
)

// handleAuthorize implements the mining.authorize command.
func handleAuthorize(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.AuthorizeCmdV1)

	fmt.Println("HERE!!!!!!!!!!!")

	if !ok {
		log.Debugf("Client %v sends an invalid mining.authorize command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	fmt.Printf("%s", cmd)

	minermgr := wsc.GetMinerManager()
	minerInfo, ok := minermgr.GetMiner(wsc.QuitChan())
	if !ok {
		log.Debugf("Client %v sends a mining.authorize command but not subscribed.", wsc.RemoteAddr())
		return nil, pooljson.ErrUnsubscribed
	}

	log.Debugf("Client %v authorized  Username is %v, password is %v.", wsc.RemoteAddr(), cmd.Username, cmd.Password)

	if cmd.Username != "" {
		_, ok = minerInfo.Username[cmd.Username]
		if ok {
			log.Debugf("Username %v already login.", cmd.Username)
			return nil, nil
		}
	}

	// Reserved field admin
	if cmd.Username == "admin" {
		log.Debugf("Client %v try to authorize with admin, reject.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	db := wsc.GetDB()
	userService := service.GetUserService()

	// Check password and address validity
	if utils.IsBlank(cmd.Password) || utils.IsBlank(cmd.Username) {
		log.Debugf("Client %v try to register without password or username.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	isExist, err := userService.UsernameExist(context.Background(), db, cmd.Username)
	if err != nil {
		log.Errorf("Database error: %v", err.Error())
		return nil, pooljson.ErrInternal
	}

	if isExist {
		success, _, err := userService.Login(context.Background(), db, cmd.Username, cmd.Password)
		if err != nil {
			log.Errorf("Login error: %v", err)
			return nil, pooljson.ErrInternal
		}
		if !success {
			log.Debugf("Password of username %v not correct.", cmd.Username)
			return nil, pooljson.ErrPasswordIncorrect
		}
	} else {
		_, err = userService.RegisterUser(context.Background(), db, cmd.Username, cmd.Password, cmd.Username, nil)
		if err != nil {
			log.Errorf("Error when register user: %v", err)
			if err == pooljson.ErrInvalidUsername || err == pooljson.ErrInvalidPassword {
				return nil, err
			}
			return nil, pooljson.ErrInternal
		}
		log.Debugf("Username %v register successfully.", cmd.Username)
	}

	minerInfo.AuthorizeAt = time.Now()
	minerInfo.LastActiveAt = time.Now()
	minerInfo.Username[cmd.Username] = time.Now()

	stopStallHandler(wsc)

	// Generate extra nonce, start notify
	errMiner := minermgr.PrepareWork(wsc.QuitChan())
	if errMiner != nil {
		return nil, errMiner
	}

	notifyClients := make(map[chan struct{}]AbstractSocketClient)
	notifyClients[wsc.QuitChan()] = wsc

	go startNotify(wsc, notifyClients, cmd.Username, minerInfo)
	go difficultyAdjustHandler(wsc, notifyClients, cmd.Username, minerInfo)

	// Add to notification map.
	ntfnMgr := wsc.GetNtfnManager()
	ntfnMgr.AddNotifyNewJobClient(wsc)

	res := true
	return &res, nil
}

func difficultyAdjustHandler(wsc AbstractSocketClient, notifyClients map[chan struct{}]AbstractSocketClient, username string, minerInfo *model.ActiveMiner) {
	defer utils.MyRecover()
	if !minerInfo.AutoDifficultyAdjust {
		return
	}

	if atomic.AddInt32(&minerInfo.DifficultyAdjustStarted, 1) != 1 {
		return
	}

	adjustTicker := time.NewTicker(constdef.DefaultDifficultyAdjustInterval * time.Second)
	defer adjustTicker.Stop()

out:
	for {
		select {
		case <-adjustTicker.C:
			username = minerInfo.GetUsername()
			userDesc := username + "(" + wsc.RemoteAddr() + ")"
			sharePerMinute := minerInfo.ShareManager.GetSharePerMinute()
			// If miner does not submit any share in the last five minutes, disconnect with it.
			if sharePerMinute == 0 && time.Now().Unix()-minerInfo.ShareManager.CreateTime.Unix() > defaultTimeout {
				log.Debugf("User %v submits 0 share in the past %v seconds, disconnecting...", defaultTimeout)
				go wsc.DisconnectGracefully()
				break out
			}

			log.Debugf("User %v submits %v share per minute", userDesc, sharePerMinute)
			if minerInfo.Difficulty == 0 {
				continue
			}
			minerInfo.EstimateHashRate(sharePerMinute, 60)
			if sharePerMinute < float64(constdef.ExpectedSharePerMinute)/float64(constdef.ShareAdjustThreshold) {
				// too slow
				if minerInfo.Difficulty == 1 {
					log.Debugf("User %v submits share too slow, cannot adjust difficulty because difficulty is already 1",
						userDesc)
					continue
				}
				log.Debugf("User %v submits share too slow, adjusting difficulty from %v to %v",
					userDesc, minerInfo.Difficulty, minerInfo.Difficulty/2)
				targetShareStr := minerInfo.SetShareDifficulty(minerInfo.Difficulty / 2)
				ntfnMgr := wsc.GetNtfnManager()
				// ntfnMgr.notifySet(notifyClients, -1, targetShareStr, "", "", -1)
				ntfnMgr.notifyDifficultyV1(notifyClients, targetShareStr)
			} else if sharePerMinute > float64(constdef.ExpectedSharePerMinute)*float64(constdef.ShareAdjustThreshold) {
				// too fast
				log.Debugf("User %v submits share too fast, adjusting difficulty from %v to %v",
					userDesc, minerInfo.Difficulty, minerInfo.Difficulty*2)
				targetShareStr := minerInfo.SetShareDifficulty(minerInfo.Difficulty * 2)
				ntfnMgr := wsc.GetNtfnManager()
				// ntfnMgr.notifySet(notifyClients, -1, targetShareStr, "", "", -1)
				ntfnMgr.notifyDifficultyV1(notifyClients, targetShareStr)
			}
		case <-wsc.QuitChan():
			break out
		}
	}
	username = minerInfo.GetUsername()
	userDesc := username + "(" + wsc.RemoteAddr() + ")"
	log.Tracef("Difficulty adjust handler for %v done", userDesc)
}

func startNotify(wsc AbstractSocketClient, notifyClients map[chan struct{}]AbstractSocketClient, username string, minerInfo *model.ActiveMiner) {
	defer utils.MyRecover()

	if atomic.AddInt32(&minerInfo.NotifyStarted, 1) != 1 {
		return
	}

	// Sleep for a few second in case the notification is sent in front of the authorization result.
	time.Sleep(time.Second * 2)

	// Notify share difficulty.
	username = minerInfo.GetUsername()
	userDesc := username + "(" + wsc.RemoteAddr() + ")"

	minerMgr := wsc.GetMinerManager()
	job := minerMgr.GetJob()
	ntfnMgr := wsc.GetNtfnManager()
	if job != nil {

		fmt.Println("WHYEEEEEEEEEEEEEEEE!")
		jobMiner := minerInfo.AddJob(job)
		log.Debugf("Sending mining.set to %v", userDesc)
		target := jobMiner.CachedTargetShareStr
		log.Debugf("Sending mining.set to %v", userDesc)

		// ntfnMgr.notifySet(notifyClients, epoch, target, "abelhash", minerInfo.ExtraNonce1, minerInfo.ExtraNonce1BitLen)

		ntfnMgr.notifyDifficultyV1(notifyClients, target)

		log.Debugf("Sending mining.notify to %v (first time)", userDesc)

		ntfnMgr.notifyNewJobV1(notifyClients, job)
		minerInfo.LastNotifyJob = job
		minerInfo.LastNotifyAt = time.Now()
	}

	fmt.Println("WHYMMMMMMMMMMMMMMMM!")
}
