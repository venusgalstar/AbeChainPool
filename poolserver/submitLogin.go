package poolserver

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/service"
	"github.com/abesuite/abe-miningpool-server/utils"
)

func handleSubmitLogin(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.EthSubmitLogin)
	if !ok {
		log.Debugf("Client %v sends an invalid eth_submitLogin command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	fmt.Printf("%s", cmd)

	log.Debugf("Client (User agent: %v) comes.", cmd.User)
	// log.Debugf("Client (User agent: %v) comes.", cmd.Params)
	log.Debugf("Client (User agent: %v) comes.", cmd.Pass)

	if hasUndesiredUserAgentV1(wsc.RemoteAddr(), cmd.User, wsc.GetAgentBlacklist(), wsc.GetAgentWhitelist()) {
		go wsc.DisconnectGracefully()
		return nil, pooljson.ErrBadUserAgent
	}

	log.Debugf("Client (User agent: %v) comes3.", cmd.Pass)

	minermgr := wsc.GetMinerManager()
	_, err := minermgr.AddNewMiner(wsc.QuitChan(), wsc.RemoteAddr(), cmd.Pass, poolProtocol, wsc.Type())
	if err != nil {
		return nil, err
	}

	log.Debugf("Client (User agent: %v) comes4.", cmd.Pass)

	if wsc.GetAbecBackendNode() == "" {
		wsc.SetAbecBackendNode(utils.GetNodeDesc())
	}

	log.Debugf("Client (User agent: %v) comes5.", cmd.Pass)

	fmt.Println("HERE!!!!!!!!!!!")

	minerInfo, ok := minermgr.GetMiner(wsc.QuitChan())
	if !ok {
		log.Debugf("Client %v sends a mining.authorize command but not subscribed.", wsc.RemoteAddr())
		return nil, pooljson.ErrUnsubscribed
	}

	log.Debugf("Client %v authorized  Username is %v, password is %v.", wsc.RemoteAddr(), cmd.User, cmd.Pass)

	if cmd.User != "" {
		_, ok = minerInfo.Username[cmd.User]
		if ok {
			log.Debugf("Username %v already login.", cmd.User)
			return nil, nil
		}
	}

	// Reserved field admin
	if cmd.User == "admin" {
		log.Debugf("Client %v try to authorize with admin, reject.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	db := wsc.GetDB()
	userService := service.GetUserService()

	isExist, err := userService.UsernameExist(context.Background(), db, cmd.User)
	if err != nil {
		log.Errorf("Database error: %v", err.Error())
		return nil, pooljson.ErrInternal
	}

	if isExist {
		success, _, err := userService.Login(context.Background(), db, cmd.User, cmd.Pass)
		if err != nil {
			log.Errorf("Login error: %v", err)
			return nil, pooljson.ErrInternal
		}
		if !success {
			log.Debugf("Password of username %v not correct.", cmd.User)
			return nil, pooljson.ErrPasswordIncorrect
		}
	} else {
		_, err = userService.RegisterUser(context.Background(), db, cmd.User, "x123456", cmd.User, nil)
		if err != nil {
			log.Errorf("Error when register user: %v", err)
			if err == pooljson.ErrInvalidUsername || err == pooljson.ErrInvalidPassword {
				return nil, err
			}
			return nil, pooljson.ErrInternal
		}
		log.Debugf("Username %v register successfully.", cmd.User)
	}

	minerInfo.AuthorizeAt = time.Now()
	minerInfo.LastActiveAt = time.Now()
	minerInfo.Username[cmd.User] = time.Now()

	stopStallHandler(wsc)

	// Generate extra nonce, start notify
	errMiner := minermgr.PrepareWork(wsc.QuitChan())
	if errMiner != nil {
		return nil, errMiner
	}

	notifyClients := make(map[chan struct{}]AbstractSocketClient)
	notifyClients[wsc.QuitChan()] = wsc

	go startNotifyE9(wsc, notifyClients, cmd.User, minerInfo)
	// go difficultyAdjustHandler(wsc, notifyClients, cmd.User, minerInfo)

	// Add to notification map.
	ntfnMgr := wsc.GetNtfnManager()
	ntfnMgr.AddNotifyNewJobClient(wsc)

	return true, nil
}

func startNotifyE9(wsc AbstractSocketClient, notifyClients map[chan struct{}]AbstractSocketClient, username string, minerInfo *model.ActiveMiner) {
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
		minerInfo.AddJob(job)
		log.Debugf("Sending mining.set to %v", userDesc)
		log.Debugf("Sending mining.set to %v", userDesc)

		log.Debugf("Sending mining.notify to %v (first time)", userDesc)

		ntfnMgr.notifyNewJobE9(notifyClients, job)
		minerInfo.LastNotifyJob = job
		minerInfo.LastNotifyAt = time.Now()
	}

	fmt.Println("WHYMMMMMMMMMMMMMMMM!")
}
