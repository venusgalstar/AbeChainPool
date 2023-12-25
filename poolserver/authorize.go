package poolserver

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/abesuite/abe-miningpool-server/chaincfg"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/constdef"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/service"
	"github.com/abesuite/abe-miningpool-server/utils"
)

// handleAuthorize implements the mining.authorize command.
func handleAuthorize(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.AuthorizeCmd)
	if !ok {
		log.Debugf("Client %v sends an invalid mining.authorize command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	minermgr := wsc.GetMinerManager()
	minerInfo, ok := minermgr.GetMiner(wsc.QuitChan())
	if !ok {
		log.Debugf("Client %v sends a mining.authorize command but not subscribed.", wsc.RemoteAddr())
		return nil, pooljson.ErrUnsubscribed
	}

	if cmd.Registering == "" {
		cmd.Registering = "0"
	}
	log.Debugf("Client %v authorized (registering: %v). Username is %v, password is %v.", wsc.RemoteAddr(), cmd.Registering, cmd.Username, cmd.Password)

	if cmd.Username != "" {
		_, ok = minerInfo.Username[cmd.Username]
		if ok {
			log.Debugf("Username %v already login.", cmd.Username)
			return nil, pooljson.ErrAlreadyAuthorized
		}
	}

	// Reserved field admin
	if cmd.Username == "admin" {
		log.Debugf("Client %v try to authorize with admin, reject.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	db := wsc.GetDB()
	userService := service.GetUserService()

	// If the user is login
	if cmd.Registering == "0" {
		// Check username validity
		if utils.IsBlank(cmd.Username) {
			log.Debugf("Client %v try to login without username.", wsc.RemoteAddr())
			return nil, pooljson.ErrInvalidRequestParams
		}
		isExist, err := userService.UsernameExist(context.Background(), db, cmd.Username)
		if err != nil {
			log.Errorf("Database error: %v", err.Error())
			return nil, pooljson.ErrInternal
		}
		if !isExist {
			log.Debugf("Client %v login with username that does not exist.", wsc.RemoteAddr())
			return nil, pooljson.ErrNotRegistered
		}
		if !utils.IsBlank(cmd.Password) {
			success, _, err := userService.Login(context.Background(), db, cmd.Username, cmd.Password)
			if err != nil {
				log.Errorf("Login error: %v", err)
				return nil, pooljson.ErrInternal
			}
			if !success {
				log.Debugf("Password of username %v not correct.", cmd.Username)
				return nil, pooljson.ErrPasswordIncorrect
			}
		}
		log.Debugf("Username %v login successfully.", cmd.Username)
	} else {
		// Register
		// Check password and address validity
		if utils.IsBlank(cmd.Password) || utils.IsBlank(cmd.Address) {
			log.Debugf("Client %v try to register without password or address.", wsc.RemoteAddr())
			return nil, pooljson.ErrInvalidRequestParams
		}
		passwordValid := utils.CheckPasswordValidity(cmd.Password)
		if !passwordValid {
			log.Debugf("Password %v is invalid", cmd.Password)
			return nil, pooljson.ErrInvalidPassword
		}
		err := utils.CheckAddressValidity(cmd.Address, chaincfg.ActiveNetParams)
		if err != nil {
			log.Debugf("Username %v address check error: %v", cmd.Username, err)
			return nil, pooljson.ErrAddressInvalid
		}

		// Calculate username
		newUsername, err := utils.CalculateUsernameFromAddress(cmd.Address)
		if err != nil {
			log.Errorf("Calculate username from address error: %v", err)
			return nil, pooljson.ErrInternal
		}
		cmd.Username = newUsername
		// Check username exist
		isExist, err := userService.UsernameExist(context.Background(), db, newUsername)
		if err != nil {
			log.Errorf("Database error: %v", err.Error())
			return nil, pooljson.ErrInternal
		}
		if isExist {
			log.Debugf("Client %v register with username that already exists.", wsc.RemoteAddr())
			return nil, pooljson.ErrAlreadyRegisteredWithUsername(newUsername)
		}
		_, err = userService.RegisterUser(context.Background(), db, newUsername, cmd.Password, cmd.Address, nil)
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
	err := minermgr.PrepareWork(wsc.QuitChan())
	if err != nil {
		return nil, err
	}

	notifyClients := make(map[chan struct{}]AbstractSocketClient)
	notifyClients[wsc.QuitChan()] = wsc

	go startNotify(wsc, notifyClients, cmd.Username, minerInfo)
	go difficultyAdjustHandler(wsc, notifyClients, cmd.Username, minerInfo)

	// Add to notification map.
	ntfnMgr := wsc.GetNtfnManager()
	ntfnMgr.AddNotifyNewJobClient(wsc)

	if cmd.Registering == "0" {
		res := pooljson.AuthorizeResult{
			Worker:     cmd.Username,
			Registered: "0",
		}
		return &res, nil
	}
	res := pooljson.AuthorizeResult{
		Worker:     cmd.Username,
		Username:   cmd.Username,
		Registered: "1",
	}
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
				ntfnMgr.notifySet(notifyClients, -1, targetShareStr, "", "", -1)
			} else if sharePerMinute > float64(constdef.ExpectedSharePerMinute)*float64(constdef.ShareAdjustThreshold) {
				// too fast
				log.Debugf("User %v submits share too fast, adjusting difficulty from %v to %v",
					userDesc, minerInfo.Difficulty, minerInfo.Difficulty*2)
				targetShareStr := minerInfo.SetShareDifficulty(minerInfo.Difficulty * 2)
				ntfnMgr := wsc.GetNtfnManager()
				ntfnMgr.notifySet(notifyClients, -1, targetShareStr, "", "", -1)
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
		jobMiner := minerInfo.AddJob(job)
		log.Debugf("Sending mining.set to %v", userDesc)
		epoch := ethash.CalculateEpoch(int32(job.Height))
		target := jobMiner.CachedTargetShareStr
		ntfnMgr.notifySet(notifyClients, epoch, target, "abelhash", minerInfo.ExtraNonce1, minerInfo.ExtraNonce1BitLen)
		log.Debugf("Sending mining.notify to %v (first time)", userDesc)
		ntfnMgr.notifyNewJob(notifyClients, job)
		minerInfo.LastNotifyJob = job
		minerInfo.LastNotifyAt = time.Now()
	}

}
