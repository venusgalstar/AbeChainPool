package main

import (
	"context"
	"fmt"
	chain "github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/dal"
	"github.com/abesuite/abe-miningpool-server/service"
	wallet "github.com/abesuite/abe-miningpool-server/walletclient"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"runtime/debug"
)

var (
	cfg *config
)

func readCAFile(filepath string) []byte {
	// Read certificate file if TLS is not disabled.
	certs, err := ioutil.ReadFile(filepath)
	if err != nil {
		poolLog.Warnf("Cannot open CA file: %v", err)
		// If there's an error reading the CA file, continue
		// with nil certs and without the client connection.
		certs = nil
	}

	return certs
}

// startChainRPC opens a RPC client connection to a abec server for blockchain
// services. This function uses the RPC options from the global config and
// there is no recovery in case the server is not available or if there is an
// authentication error. Instead, all requests to the client will simply error.
func startChainRPC(certs []byte) (*chain.RPCClient, error) {
	poolLog.Infof("Attempting RPC client connection to Abec (%v)", cfg.RPCConnect)
	rpcc, err := chain.NewRPCClient(cfg.RPCConnect, cfg.RPCUser, cfg.RPCPass, certs, cfg.DisableClientTLS,
		0, cfg.UseAddrFromAbec, cfg.MiningAddrs)
	if err != nil {
		return nil, err
	}
	err = rpcc.Start()
	if err := rpcc.NotifyBlocks(); err != nil {
		log.Fatal(err)
	}
	return rpcc, err
}

func startWalletRPC(certs []byte) (*wallet.RPCClient, error) {
	poolLog.Infof("Attempting RPC client connection to Abewallet (%v)", cfg.WalletRPCConnect)
	walletRPCClient, err := wallet.NewRPCClient(cfg.WalletRPCConnect, cfg.WalletRPCUser, cfg.WalletRPCPass, certs, cfg.DisableWalletClientTLS, 0)
	if err != nil {
		return nil, err
	}
	err = walletRPCClient.Start()
	if err := walletRPCClient.NotifyTransactions(); err != nil {
		log.Fatal(err)
	}
	return walletRPCClient, err
}

func chainRPCClientConnectLoop(cfg *config, rpc *chainClients) {
	var certs []byte
	if !cfg.DisableClientTLS {
		certs = readCAFile(cfg.CAFile.Value)
	}

	for {
		var (
			chainClient *chain.RPCClient
			err         error
		)

		chainClient, err = startChainRPC(certs)
		if err != nil {
			poolLog.Errorf("Unable to open connection to consensus RPC server: %v", err)
			continue
		}

		rpc.setChainClient(chainClient)

		chainClient.WaitForShutdown()
		break
	}
}

func walletRPCClientLoop(cfg *config, walletPostClient *walletClient) {
	var certs []byte
	if !cfg.DisableWalletClientTLS {
		certs = readCAFile(cfg.WalletCAFile.Value)
	}

	// For Test Connection Of Wallet
	//message, err := walletRPCClient.GetBalancesabeAsync().Receive()
	//if err != nil {
	//	poolLog.Errorf("Unable to open connection to wallet RPC server: %v", err)
	//}
	//poolLog.Infof("%v", message)
	for {

		var (
			walletRPCClient *wallet.RPCClient
			err             error
		)
		walletRPCClient, err = startWalletRPC(certs)
		if err != nil {
			poolLog.Errorf("Unable to open connection to wallet RPC server: %v", err)
		}

		if walletRPCClient == nil || walletRPCClient.Client == nil {
			return
		}

		walletPostClient.setWalletClient(walletRPCClient)
		walletRPCClient.WaitForShutdown()
		break
	}
}

func calculateRewardIntervalChangeHeight(latestHeight int) int {
	currentEpoch := ethash.CalculateEpoch(int32(latestHeight))
	return ethash.CalculateEpochStartHeight(currentEpoch + 1)
}

func startProfileServer() {
	listenAddr := net.JoinHostPort("localhost", cfg.ProfilePort)
	poolLog.Infof("Profile server listening on %s", listenAddr)
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	poolLog.Errorf("%v", http.ListenAndServe(listenAddr, mux))
}

func poolMain() error {

	// Load configuration and parse command line. This function also
	// initializes logging and configures it accordingly.
	tcfg, _, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = tcfg

	defer func() {
		if logRotator != nil {
			logRotator.Close()
		}
	}()

	defer poolLog.Info("Shutdown complete")

	// Enable http profiling server if requested.
	if cfg.ProfilePort != "" {
		go func() {
			startProfileServer()
		}()
	}

	// initiate database
	err = dal.InitDB(&dal.DBConfig{
		Username:     cfg.DbUsername,
		Password:     cfg.DbPassword,
		Address:      cfg.DbAddress,
		DatabaseName: cfg.DbName,
	}, !cfg.DisableAutoCreateDB)
	if err != nil {
		return err
	}

	// check if admin exist
	ctx := context.Background()
	tx := dal.GetDB(ctx)
	userService := service.GetUserService()
	exist, err := userService.UsernameExist(ctx, tx, "admin")
	if err != nil {
		return err
	}
	if !exist {
		fmt.Print("It seems that it is the first time you start the server, please specify the password for admin: ")
		var pwd string
		_, err := fmt.Scanln(&pwd)
		if err != nil {
			return err
		}
		err = userService.RegisterAdmin(ctx, tx, pwd)
		if err != nil {
			return err
		}
		poolLog.Infof("Successfully generate account for admin, please use this password for RPC authentication")
	}

	// show meta info
	metaInfoService := service.GetMetaInfoService()
	metaInfo, err := metaInfoService.Get(ctx, tx)
	if err != nil {
		return err
	}
	poolLog.Infof("Meta Info: Balance: %v, Last reward height: %v, Last transaction confirmed height %v", metaInfo.Balance, metaInfo.LastRewardHeight, metaInfo.LastTxCheckHeight)

	// get previous config
	configService := service.GetConfigInfoService()
	latestConfig, err := configService.GetLatestConfig(ctx, tx)
	if err != nil {
		return err
	}
	latestUserShareInfo, err := userService.GetLatestUserShareInfo(ctx, tx)
	if err != nil {
		return err
	}
	if latestConfig == nil {
		poolLog.Info("No previous config info found.")
		if latestUserShareInfo == nil {
			cfg.RewardIntervalPre = cfg.RewardInterval
		} else {
			cfg.RewardIntervalPre = int(latestUserShareInfo.EndHeight - latestUserShareInfo.StartHeight)
			if cfg.RewardIntervalPre != cfg.RewardInterval {
				cfg.ConfigChangeHeight = calculateRewardIntervalChangeHeight(int(latestUserShareInfo.EndHeight - 1))
				poolLog.Infof("Reward interval changes (previous: %v new: %v), will take effect in the next epoch (height: %v)",
					cfg.RewardIntervalPre, cfg.RewardInterval, cfg.ConfigChangeHeight)
			}
		}
	} else {
		cfg.RewardIntervalPre = int(latestConfig.RewardInterval)
		if cfg.RewardIntervalPre != cfg.RewardInterval {
			cfg.ConfigChangeHeight = int(latestConfig.EndHeight)
			poolLog.Infof("Reward interval changes (previous: %v new: %v), will take effect in the next epoch (height: %v)",
				cfg.RewardIntervalPre, cfg.RewardInterval, cfg.ConfigChangeHeight)
		}
		cfg.ManageFeePercentPre = latestConfig.ManageFeePercent
		if cfg.ManageFeePercentPre != cfg.ManageFeePercent {
			cfg.ConfigChangeHeight = int(latestConfig.EndHeight)
			poolLog.Infof("Manage fee changes (previous: %v%% new: %v%%), will take effect in the next epoch (height: %v)",
				cfg.ManageFeePercentPre, cfg.ManageFeePercent, cfg.ConfigChangeHeight)
		}
	}

	// create a container for chain clients
	rpc, err := createChainClient(cfg)
	if err != nil {
		poolLog.Errorf("Unable to create chain RPC client for Abec: %v", err)
		return err
	}

	var walletPostClient *walletClient
	if cfg.UseWallet {
		walletPostClient, err = createWalletClient(cfg)
		if err != nil {
			poolLog.Errorf("Unable to create chain RPC client for Abewallet: %v", err)
			return err
		}
	}

	// create and start rpc chain client
	if !cfg.DisableConnectToAbec {
		go chainRPCClientConnectLoop(cfg, rpc)
	}
	if cfg.UseWallet {
		go walletRPCClientLoop(cfg, walletPostClient)
	}

	// create and start server, including pool rpc server and miner manager
	svr, err := newServer(rpc, walletPostClient)
	if err != nil {
		return err
	}

	svr.Start()

	if rpc != nil {
		addInterruptHandler(func() {
			rpc.Stop()
		})
	}
	if walletPostClient != nil {
		addInterruptHandler(func() {
			walletPostClient.Stop()
		})
	}
	if svr != nil {
		addInterruptHandler(func() {
			svr.Stop()
		})
	}

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interruptHandlersDone
	return nil
}

func main() {
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Block and transaction processing can cause bursty allocations.  This
	// limits the garbage collector from excessively overallocating during
	// bursts.  This value was arrived at with the help of profiling live
	// usage.
	debug.SetGCPercent(10)

	if err := poolMain(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// DistributeBonus constructs and sends a legacy sendto request which
// contains two separate bools to denote verbosity, in contract to a single int
// parameter.
