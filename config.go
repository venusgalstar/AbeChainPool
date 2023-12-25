package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/abesuite/abe-miningpool-server/chaincfg"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/utils"

	flags "github.com/jessevdk/go-flags"
)

const (
	defaultConfigFilename       = "miningpool-server.conf"
	defaultLogDirname           = "logs"
	defaultLogFilename          = "abe-miningpool-server.log"
	defaultDbType               = "mysql"
	sampleConfigFilename        = "miningpool-server.conf"
	defaultLogLevel             = "info"
	defaultListenerPort         = "27777"
	defaultListenerPortSocket   = "27778"
	defaultPoolLimitUser        = "miner"
	defaultPoolLimitPass        = "miner"
	defaultMaxRPCClients        = 10000
	defaultMaxRPCWebsockets     = 10000
	defaultMaxRPCConcurrentReqs = 200
	defaultDbAddress            = "127.0.0.1:3306"
	defaultManageFeePercent     = 20
	defaultTxFeeSpecified       = 1000000
	defaultRewardMaturity       = 200
	defaultRewardIntervals      = 100
	defaultDelayIntervals       = 50
	defaultDatabaseName         = "abe_mining_pool"
	defaultMaxErrors            = 10
	defaultExtraNonce2BitLen    = 48
)

var (
	abecDefaultCAFile         = filepath.Join(utils.AppDataDir("abec", false), "rpc.cert")
	abewalletDefaultCAFile    = filepath.Join(utils.AppDataDir("abewallet", false), "rpc.cert")
	defaultHomeDir            = utils.AppDataDir("abe-miningpool-server", false)
	localConfigFile           = defaultConfigFilename
	knownDbTypes              = []string{"mysql"}
	localAbecRPCCertFile      = "abec.cert"
	localAbewalletRPCCertFile = "abewallet.cert"
	localPoolKeyFile          = "pool.key"
	localPoolCertFile         = "pool.cert"
	defaultLogDir             = filepath.Join(defaultHomeDir, defaultLogDirname)
	netParams                 = &chaincfg.MainNetParams
)

// config defines the configuration options for mining pool.
//
// See loadConfig for details on the configuration load process.
type config struct {
	AdminBlacklists       []string `long:"adminblacklist" description:"Add an IP network or IP that will be banned if an admin request the rpc server. (eg. 192.168.1.0/24 or ::1)"`
	adminBlacklists       []*net.IPNet
	AdminWhitelists       []string `long:"adminwhitelist" description:"Add an IP network or IP that will not be banned if an admin request the rpc server. (eg. 192.168.1.0/24 or ::1)"`
	adminWhitelists       []*net.IPNet
	AgentBlacklist        []string              `long:"agentblacklist" description:"A comma separated list of user-agent substrings which will cause mining pool to reject any clients whose user-agent contains any of the blacklisted substrings."`
	AgentWhitelist        []string              `long:"agentwhitelist" description:"A comma separated list of user-agent substrings which will cause mining pool to require all clients' user-agents to contain one of the whitelisted substrings. The blacklist is applied before the blacklist, and an empty whitelist will allow all agents that do not fail the blacklist."`
	AppDataDir            *utils.ExplicitString `short:"A" long:"appdata" description:"Application data directory for mining pool config and logs"`
	Blacklists            []string              `long:"blacklist" description:"Add an IP network or IP that will be banned. (eg. 192.168.1.0/24 or ::1)"`
	blacklists            []*net.IPNet
	CAFile                *utils.ExplicitString `long:"cafile" description:"File containing root certificates to authenticate a TLS connections with abec"`
	ConfigFile            string                `short:"C" long:"configfile" description:"Path to configuration file"`
	DbType                string                `long:"dbtype" description:"Database backend to use for the data"`
	DbUsername            string                `long:"dbusername" description:"username which is used to connect with database"`
	DbPassword            string                `long:"dbpassword" description:"password which is used to connect with database"`
	DbAddress             string                `long:"dbaddress" description:"ip address and port of database (default: 127.0.0.1:3306)"`
	DetailedShareInfo     bool                  `long:"detailedshareinfo" description:"Record detailed share info including share hash and seal hash"`
	DbName                string                `long:"dbname" description:"name of server database (default: abe_mining_pool)"`
	ExternalIPs           []string              `long:"externalip" description:"Add an ip to the list of local addresses we claim to listen on to clients (miners)"`
	ExtraNonce2BitLen     int                   `long:"extranonce2bitlen" description:"The bit length of extra nonce 2, which means the probing range of miner"`
	Listeners             []string              `long:"listen" description:"Add an interface/port to listen for connections (HTTP/ws)"`
	ListenersTCPSocket    []string              `long:"listenertcpsocket" description:"Add an interface/port to listen for connections (TCP socket)"`
	ListenerPort          string                `long:"listenerport" description:"listenerport is the port that HTTP/ws server listen on (default: 27777)"`
	ListenerPortTCPSocket string                `long:"listenerporttcpsocket" description:"listenerport is the port that TCP socket server listen on (default: 27778)"`
	LogDir                string                `long:"logdir" description:"Directory to log output."`
	// todo: the following three config should be reconsidered
	RPCMaxClients               int  `long:"rpcmaxclients" description:"Max number of rpc clients (miners and admin)"`
	RPCMaxConcurrentReqs        int  `long:"rpcmaxconcurrentreqs" description:"Max number of concurrent RPC requests that may be processed concurrently"`
	RPCMaxWebsockets            int  `long:"rpcmaxwebsockets" description:"Max number of websocket connections (miners)"`
	DisableAutoCreateDB         bool `long:"noautocreatedb" description:"Disable creating database and table automatically"`
	DisableAutoAllocate         bool `long:"noautoallocate" description:"Disable auto allocate rewards"`
	DisableAutoDifficultyAdjust bool `long:"noautodifficultyadjust" description:"Disable auto difficulty adjust"`
	// todo: not implemented yet
	DisableBanning         bool     `long:"nobanning" description:"Disable banning of misbehaving miners"`
	DisableTLS             bool     `long:"notls" description:"Disable TLS for the websocket RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost"`
	DisableTCPSocketTLS    bool     `long:"notcpsockettls" description:"Disable TLS for the TCP socket RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost"`
	DisableClientTLS       bool     `long:"noclienttls" description:"Disable TLS for the connection with abec"`
	DisableWalletClientTLS bool     `long:"nowalletclienttls" description:"Disable TLS for the connection with abewallet"`
	DebugLevel             string   `short:"d" long:"debuglevel" description:"Logging level for all subsystems {trace, debug, info, warn, error, critical} -- You may also specify <subsystem>=<level>,<subsystem2>=<level>,... to set the log level for individual subsystems -- Use show to list available subsystems"`
	EnableTCPSocket        bool     `long:"enabletcpsocket" description:"Enable TCP socket RPC server"`
	RewardInterval         int      `long:"rewardinterval" description:"Number of blocks between allocations of rewards (default: 100)"`
	DelayInterval          int      `long:"delayinterval" description:"Number of blocks between allocations of rewards with database and create transaction  (default: 50)"`
	RewardMaturity         int      `long:"rewardmaturity" description:"Number of blocks before the reward mature (which can be allocated to miners) (default: 50)"`
	TxFeeSpecified         float64  `long:"txfeespecified" description:"txfeespecified specifies the transaction fee when allocate reward. (default: 100)"`
	RegressionTest         bool     `long:"regtest" description:"Use the regression test network"`
	RPCConnect             string   `short:"c" long:"rpcconnect" description:"Hostname/IP and port of abec RPC server to connect to"`
	RPCPass                string   `short:"P" long:"abecrpcpass" default-mask:"-" description:"Password for RPC connections with abec"`
	RPCUser                string   `short:"u" long:"abecrpcuser" description:"Username for RPC connections with abec"`
	PoolUser               string   `long:"pooluser" description:"RPC username for pool admin, this is used to control the mining pool (default: admin). This should be changed in production environment"`
	PoolPass               string   `long:"poolpass" description:"RPC password for pool admin, this is used to control the mining pool (default: admin). This should be changed in production environment"`
	PoolLimitUser          string   `long:"poollimituser" description:"RPC username for individual miner, this is used for miners to establish RPC websocket connection with the mining pool"`
	PoolLimitPass          string   `long:"poollimitpass" description:"RPC password for individual miner, this is used for miners to establish RPC websocket connection with the mining pool"`
	PoolCert               string   `long:"poolcert" description:"File containing the certificate file for users to connect with mining pool"`
	PoolKey                string   `long:"poolkey" description:"File containing the certificate key for users to connect with mining pool"`
	ProfilePort            string   `long:"profileport" description:"Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65536"`
	Upnp                   bool     `long:"upnp" description:"Use UPnP to map our listening port outside of NAT"`
	UseAddrFromAbec        bool     `long:"useaddrfromabec" description:"Use mining address provided by abec"`
	ShowVersion            bool     `short:"V" long:"version" description:"Display version information and exit"`
	SimNet                 bool     `long:"simnet" description:"Use the simulation test network"`
	DisableConnectToAbec   bool     `long:"noconnecttoabec" description:"Do not connect to abec"`
	UseWallet              bool     `long:"usewallet" description:"Connect to wallet to send transactions"`
	TestNet                bool     `long:"testnet" description:"Use the test network"`
	ManageFeePercent       int      `long:"managefeepercent" description:"The pool management fee, for example, if manage fee percent is 20, then 20% of coinbase reward will be given to mining pool (default: 20)"`
	MaxErrors              int      `long:"maxerrors" description:"Max errors before disconnecting with miner"`
	MiningAddrs            []string `long:"miningaddr" description:"Add the specified payment address to the list of addresses to use for generated blocks -- At least one address is required if the generate or externalgenerate option is set"`
	ServerManageFeeAddrs   []string `long:"servermanagefeeaddr" description:"The addresses to use when sending management fee"`

	WalletCAFile     *utils.ExplicitString `long:"walletcafile" description:"File containing root certificates to authenticate a TLS connections with abewallet"`
	WalletRPCConnect string                `long:"walletrpcconnect" description:"Hostname/IP and port of abewallet RPC server to connect to"`
	WalletRPCUser    string                `long:"walletrpcuser" description:"Username for RPC connections with abewallet"`
	WalletRPCPass    string                `long:"walletrpcpass" default-mask:"-" description:"Password for RPC connections with abewallet"`
	Whitelists       []string              `long:"whitelist" description:"Add an IP network or IP that will not be banned. (eg. 192.168.1.0/24 or ::1)"`
	whitelists       []*net.IPNet

	EthashConfig          ethash.Config
	ethashVerifyByFullDAG bool   `long:"ethashVerifyByFullDAG" description:"Use full DAG to verify EthashPow"`
	DAGDir                string `long:"dagdir" description:"Directory that store DAG data"`
	WorkingDir            string `long:"workingdir" description:"Working directory"`
	RewardIntervalPre     int
	ConfigChangeHeight    int
	ManageFeePercentPre   int
}

// newConfigParser returns a new command line flags parser.
func newConfigParser(cfg *config, options flags.Options) *flags.Parser {
	parser := flags.NewParser(cfg, options)
	return parser
}

// createDefaultConfig copies the file sample-miningpool-server.conf to the given destination path,
// and populates it with some randomly generated RPC username and password.
func createDefaultConfigFile(destinationPath string) error {
	// Create the destination directory if it does not exists
	err := os.MkdirAll(filepath.Dir(destinationPath), 0700)
	if err != nil {
		return err
	}

	// We assume sample config file path is same as binary
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	sampleConfigPath := filepath.Join(path, sampleConfigFilename)

	// We generate a random user and password
	randomBytes := make([]byte, 20)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return err
	}
	generatedRPCUser := base64.StdEncoding.EncodeToString(randomBytes)

	_, err = rand.Read(randomBytes)
	if err != nil {
		return err
	}
	generatedRPCPass := base64.StdEncoding.EncodeToString(randomBytes)

	src, err := os.Open(sampleConfigPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.OpenFile(destinationPath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dest.Close()

	// We copy every line from the sample config file to the destination,
	// only replacing the two lines for rpcuser and rpcpass
	reader := bufio.NewReader(src)
	for err != io.EOF {
		var line string
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		if strings.Contains(line, "rpcuser=") {
			line = "rpcuser=" + generatedRPCUser + "\n"
		} else if strings.Contains(line, "rpcpass=") {
			line = "rpcpass=" + generatedRPCPass + "\n"
		}

		if _, err := dest.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(defaultHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but the variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

// supportedSubsystems returns a sorted slice of the supported subsystems for
// logging purposes.
func supportedSubsystems() []string {
	// Convert the subsystemLoggers map keys to a slice.
	subsystems := make([]string, 0, len(subsystemLoggers))
	for subsysID := range subsystemLoggers {
		subsystems = append(subsystems, subsysID)
	}

	// Sort the subsystems for stable display.
	sort.Strings(subsystems)
	return subsystems
}

// filesExists reports whether the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// validLogLevel returns whether or not logLevel is a valid debug log level.
func validLogLevel(logLevel string) bool {
	switch logLevel {
	case "trace":
		fallthrough
	case "debug":
		fallthrough
	case "info":
		fallthrough
	case "warn":
		fallthrough
	case "error":
		fallthrough
	case "critical":
		return true
	}
	return false
}

// validDbType returns whether or not dbType is a supported database type.
func validDbType(dbType string) bool {
	for _, knownType := range knownDbTypes {
		if dbType == knownType {
			return true
		}
	}

	return false
}

// parseAndSetDebugLevels attempts to parse the specified debug level and set
// the levels accordingly.  An appropriate error is returned if anything is
// invalid.
func parseAndSetDebugLevels(debugLevel string) error {
	// When the specified string doesn't have any delimters, treat it as
	// the log level for all subsystems.
	if !strings.Contains(debugLevel, ",") && !strings.Contains(debugLevel, "=") {
		// Validate debug log level.
		if !validLogLevel(debugLevel) {
			str := "The specified debug level [%v] is invalid"
			return fmt.Errorf(str, debugLevel)
		}

		// Change the logging level for all subsystems.
		setLogLevels(debugLevel)

		return nil
	}

	// Split the specified string into subsystem/level pairs while detecting
	// issues and update the log levels accordingly.
	for _, logLevelPair := range strings.Split(debugLevel, ",") {
		if !strings.Contains(logLevelPair, "=") {
			str := "The specified debug level contains an invalid " +
				"subsystem/level pair [%v]"
			return fmt.Errorf(str, logLevelPair)
		}

		// Extract the specified subsystem and log level.
		fields := strings.Split(logLevelPair, "=")
		subsysID, logLevel := fields[0], fields[1]

		// Validate subsystem.
		if _, exists := subsystemLoggers[subsysID]; !exists {
			str := "The specified subsystem [%v] is invalid -- " +
				"supported subsytems %v"
			return fmt.Errorf(str, subsysID, supportedSubsystems())
		}

		// Validate log level.
		if !validLogLevel(logLevel) {
			str := "The specified debug level [%v] is invalid"
			return fmt.Errorf(str, logLevel)
		}

		setLogLevel(subsysID, logLevel)
	}

	return nil
}

// removeDuplicateAddresses returns a new slice with all duplicate entries in
// addrs removed.
func removeDuplicateAddresses(addrs []string) []string {
	result := make([]string, 0, len(addrs))
	seen := map[string]struct{}{}
	for _, val := range addrs {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = struct{}{}
		}
	}
	return result
}

// normalizeAddress returns addr with the passed default port appended if
// there is not already a port specified.
func normalizeAddress(addr, defaultPort string) string {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		return net.JoinHostPort(addr, defaultPort)
	}
	return addr
}

// genCertPair generates a key/cert pair to the paths provided.
func genCertPair(certFile string, keyFile string, externalIPs []string) error {
	poolLog.Infof("Generating TLS certificates of mining pool...")

	org := "abe mining pool autogenerated cert"
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)

	extraHosts := externalIPs
	serverIP := ""
	fmt.Print("Please enter the ip address of your server (press enter if you just test on local area network): ")
	_, err := fmt.Scanln(&serverIP)
	if serverIP != "" {
		extraHosts = append(extraHosts, serverIP)
	}

	cert, key, err := utils.NewTLSCertPair(org, validUntil, extraHosts)
	if err != nil {
		return err
	}

	// Write cert and key files.
	if err = ioutil.WriteFile(certFile, cert, 0666); err != nil {
		return err
	}
	if err = ioutil.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		return err
	}

	poolLog.Infof("Done generating TLS certificates")
	return nil
}

// normalizeAddresses returns a new slice with all the passed peer addresses
// normalized with the given default port, and all duplicates removed.
func normalizeAddresses(addrs []string, defaultPort string) []string {
	for i, addr := range addrs {
		addrs[i] = normalizeAddress(addr, defaultPort)
	}

	return removeDuplicateAddresses(addrs)
}

func checkRewardIntervals(rewardInterval int) bool {
	if rewardInterval <= 0 {
		return false
	}
	if int(chaincfg.ActiveNetParams.EthashEpochLength)%rewardInterval != 0 {
		return false
	}
	return true
}

func loadConfig() (*config, []string, error) {
	cfg := config{
		ConfigFile:           localConfigFile,
		CAFile:               utils.NewExplicitString(""),
		WalletCAFile:         utils.NewExplicitString(""),
		AppDataDir:           utils.NewExplicitString(defaultHomeDir),
		DebugLevel:           defaultLogLevel,
		LogDir:               defaultLogDir,
		RPCMaxClients:        defaultMaxRPCClients,
		RPCMaxConcurrentReqs: defaultMaxRPCConcurrentReqs,
		RPCMaxWebsockets:     defaultMaxRPCWebsockets,
		DbType:               defaultDbType,
		PoolKey:              localPoolKeyFile,
		PoolCert:             localPoolCertFile,
		EthashConfig:         ethash.DefaultCfg,
		MaxErrors:            defaultMaxErrors,
		ConfigChangeHeight:   -1,
		ExtraNonce2BitLen:    defaultExtraNonce2BitLen,
		DbName:               defaultDatabaseName,
	}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.
	preCfg := cfg
	preParser := newConfigParser(&preCfg, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
			return nil, nil, err
		}
	}

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)
	if preCfg.ShowVersion {
		fmt.Println(appName, "version", version()) //this process is all in version.go
		os.Exit(0)
	}

	if preCfg.WorkingDir != "" {
		err := os.Chdir(preCfg.WorkingDir)
		if err != nil {
			return nil, nil, err
		}
	}

	fmt.Printf("Use config file: %v\n", preCfg.ConfigFile)

	// Load additional config from file.
	var configFileError error
	parser := newConfigParser(&cfg, flags.Default)
	if _, err := os.Stat(preCfg.ConfigFile); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("cannot find config file %v", preCfg.ConfigFile)
	}

	err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config "+
				"file: %v\n", err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, nil, err
		}
		configFileError = err
	}

	// Parse command line options again to ensure they take precedence.
	remainingArgs, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			fmt.Fprintln(os.Stderr, usageMessage)
		}
		return nil, nil, err
	}

	// Create the home directory if it doesn't already exist.
	funcName := "loadConfig"
	err = os.MkdirAll(defaultHomeDir, 0700)
	if err != nil {
		// Show a nicer error message if it's because a symlink is
		// linked to a directory that does not exist (probably because
		// it's not mounted).
		if e, ok := err.(*os.PathError); ok && os.IsExist(err) {
			if link, lerr := os.Readlink(e.Path); lerr == nil {
				str := "is symlink %s -> %s mounted?"
				err = fmt.Errorf(str, e.Path, link)
			}
		}

		str := "%s: Failed to create home directory: %v"
		err := fmt.Errorf(str, funcName, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, err
	}

	// Expand the log directory
	cfg.LogDir = cleanAndExpandPath(cfg.LogDir)

	// Special show command to list supported subsystems and exit.
	if cfg.DebugLevel == "show" {
		fmt.Println("Supported subsystems", supportedSubsystems())
		os.Exit(0)
	}

	// Initialize log rotation. After log rotation has been initialized, the
	// logger variables may be used.
	initLogRotator(filepath.Join(cfg.LogDir, defaultLogFilename))

	// Parse, validate, and set debug log level(s).
	if err := parseAndSetDebugLevels(cfg.DebugLevel); err != nil {
		err := fmt.Errorf("%s: %v", funcName, err.Error())
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, nil, err
	}

	// Show version at startup.
	poolLog.Infof("Version %s", version())

	// Validate database type.
	if !validDbType(cfg.DbType) {
		str := "%s: The specified database type [%v] is invalid -- " +
			"supported types %v"
		err := fmt.Errorf(str, funcName, cfg.DbType, knownDbTypes)
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, nil, err
	}

	// Set DAG directory
	if cfg.DAGDir != "" {
		if err := os.MkdirAll(cfg.DAGDir, 0755); err != nil {
			return nil, nil, err
		}
		cfg.EthashConfig.DatasetDir = cfg.DAGDir
		cfg.EthashConfig.CacheDir = cfg.DAGDir
	}

	// Multiple networks can't be selected simultaneously.
	numNets := 0
	// Count number of network flags passed; assign active network params
	// while we're at it
	if cfg.TestNet {
		numNets++
		netParams = &chaincfg.TestNet3Params
		cfg.DbName = cfg.DbName + "_" + netParams.Name
	}
	if cfg.RegressionTest {
		numNets++
		netParams = &chaincfg.RegNetParams
		cfg.DbName = cfg.DbName + "_" + netParams.Name
	}
	if cfg.SimNet {
		numNets++
		netParams = &chaincfg.SimNetParams
		cfg.DbName = cfg.DbName + "_" + netParams.Name
	}
	if numNets > 1 {
		str := "%s: The testnet, regtest, and simnet params " +
			"can't be used together -- choose one of the three"
		err := fmt.Errorf(str, funcName)
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, nil, err
	}

	if cfg.RegressionTest {
		return nil, nil, errors.New("regression test is not supported by mining pool, please use mainnet or testnet or simnet")
	}

	chaincfg.ActiveNetParams = netParams

	// Validate any given whitelisted IP addresses and networks.
	if len(cfg.Whitelists) > 0 {
		var ip net.IP
		cfg.whitelists = make([]*net.IPNet, 0, len(cfg.Whitelists))

		for _, addr := range cfg.Whitelists {
			_, ipnet, err := net.ParseCIDR(addr)
			if err != nil {
				ip = net.ParseIP(addr)
				if ip == nil {
					str := "%s: The whitelist value of '%s' is invalid"
					err = fmt.Errorf(str, funcName, addr)
					fmt.Fprintln(os.Stderr, err)
					fmt.Fprintln(os.Stderr, usageMessage)
					return nil, nil, err
				}
				var bits int
				if ip.To4() == nil {
					// IPv6
					bits = 128
				} else {
					bits = 32
				}
				ipnet = &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(bits, bits),
				}
			}
			cfg.whitelists = append(cfg.whitelists, ipnet)
		}
	}

	if cfg.ListenerPort == "" {
		cfg.ListenerPort = defaultListenerPort
	}
	// Add the default listener if none were specified. The default
	// listener is all addresses on the listen port for the network
	// we are to connect to.
	if len(cfg.Listeners) == 0 {
		cfg.Listeners = []string{
			net.JoinHostPort("", cfg.ListenerPort),
		}
	}

	if cfg.EnableTCPSocket {
		poolLog.Infof("Enable TCP socket server")
		if cfg.ListenerPortTCPSocket == "" {
			cfg.ListenerPortTCPSocket = defaultListenerPortSocket
		}
		if len(cfg.ListenersTCPSocket) == 0 {
			cfg.ListenersTCPSocket = []string{
				net.JoinHostPort("", cfg.ListenerPortTCPSocket),
			}
		}
	}

	// Add default port to all listener addresses if needed and remove
	// duplicate addresses.
	cfg.Listeners = normalizeAddresses(cfg.Listeners, cfg.ListenerPort)

	// If do not connect to abec, also disable connect to wallet
	if cfg.DisableConnectToAbec {
		cfg.UseWallet = false
	}

	// If no username or password is provided, we can not connect to abec
	if cfg.RPCUser == "" || cfg.RPCPass == "" {
		return nil, nil, errors.New("abecrpcuser and abecrpcpass should be configured to connect with abec")
	}

	// Add default username and password for individual miner
	if cfg.PoolLimitUser == "" || cfg.PoolLimitPass == "" {
		cfg.PoolLimitUser = defaultPoolLimitUser
		cfg.PoolLimitPass = defaultPoolLimitPass
	}

	if cfg.RPCConnect == "" {
		cfg.RPCConnect = net.JoinHostPort("localhost", netParams.RPCClientPort)
	}
	// Add default port to connect flag if missing.
	cfg.RPCConnect, err = utils.NormalizeAddress(cfg.RPCConnect, netParams.RPCClientPort)
	if err != nil {
		return nil, nil, err
	}

	RPCHost, _, err := net.SplitHostPort(cfg.RPCConnect)
	if err != nil {
		return nil, nil, err
	}

	localhostListeners := map[string]struct{}{
		"localhost": {},
		"127.0.0.1": {},
		"::1":       {},
	}

	// If CAFile is unset, choose either the copy or local abec cert.
	if !cfg.DisableClientTLS && !cfg.CAFile.ExplicitlySet() {
		cfg.CAFile.Value = localAbecRPCCertFile

		// If the CA copy does not exist, check if we're connecting to
		// a local abec and switch to its RPC cert if it exists.
		certExists, err := utils.FileExists(cfg.CAFile.Value)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return nil, nil, err
		}
		if !certExists {
			if _, ok := localhostListeners[RPCHost]; ok {
				abecCertExists, err := utils.FileExists(
					abecDefaultCAFile)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return nil, nil, err
				}
				if abecCertExists {
					cfg.CAFile.Value = abecDefaultCAFile
				}
			}
		}
	}

	if cfg.UseWallet {
		// If no username or password is provided, we can not connect to abewallet
		if cfg.WalletRPCUser == "" || cfg.WalletRPCPass == "" {
			return nil, nil, errors.New("walletrpcuser and walletrpcpass should be configured to communicate with abewallet")
		}

		if cfg.WalletRPCConnect == "" {
			if netParams == &chaincfg.RegNetParams {
				return nil, nil, errors.New("wallet not yet compatible with regtest")
			}
			cfg.WalletRPCConnect = net.JoinHostPort("localhost", netParams.WalletRPCClientPort)
		}
		// Add default port to connect flag if missing.
		cfg.WalletRPCConnect, err = utils.NormalizeAddress(cfg.WalletRPCConnect, netParams.WalletRPCClientPort)
		if err != nil {
			return nil, nil, err
		}

		walletHost, _, err := net.SplitHostPort(cfg.WalletRPCConnect)
		if err != nil {
			return nil, nil, err
		}

		// If WalletCAFile is unset, choose either the copy or local abewallet cert.
		if !cfg.DisableWalletClientTLS && !cfg.WalletCAFile.ExplicitlySet() {
			cfg.WalletCAFile.Value = localAbewalletRPCCertFile

			// If the CA copy does not exist, check if we're connecting to
			// a local abewallet and switch to its RPC cert if it exists.
			walletCertExists, err := utils.FileExists(cfg.WalletCAFile.Value)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return nil, nil, err
			}
			if !walletCertExists {
				if _, ok := localhostListeners[walletHost]; ok {
					abewalletCertExists, err := utils.FileExists(
						abewalletDefaultCAFile)
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						return nil, nil, err
					}
					if abewalletCertExists {
						cfg.WalletCAFile.Value = abewalletDefaultCAFile
					}
				}
			}
		}

		if cfg.DelayInterval <= 0 {
			cfg.DelayInterval = defaultDelayIntervals
		}
		poolLog.Infof("Delay interval: %v", cfg.DelayInterval)

		if cfg.TxFeeSpecified <= 0 {
			cfg.TxFeeSpecified = defaultTxFeeSpecified
		}
	} else {
		poolLog.Infof("Disable wallet")
	}

	if cfg.DbUsername == "" || cfg.DbPassword == "" {
		return nil, nil, errors.New("database username or password not configured, please add them in configuration file or " +
			"specify them using --dbusername and --dbpassword")
	}

	if cfg.DbAddress == "" {
		poolLog.Infof("Use default database address: %v", defaultDbAddress)
		cfg.DbAddress = defaultDbAddress
	}

	if cfg.DbName == "" {
		return nil, nil, fmt.Errorf("nil dbname")
	}

	if cfg.RewardInterval <= 0 {
		cfg.RewardInterval = defaultRewardIntervals
	}
	poolLog.Infof("Reward per blocks: %v", cfg.RewardInterval)

	isValidInterval := checkRewardIntervals(cfg.RewardInterval)
	if !isValidInterval {
		errStr := "Invalid reward interval, only 100, 200, 400, 500, 800, 1000, 2000, 4000 is allowed"
		return nil, nil, errors.New(errStr)
	}

	if cfg.RewardMaturity <= 0 {
		cfg.RewardMaturity = defaultRewardMaturity
	}
	poolLog.Infof("Reward maturity: %v", cfg.RewardMaturity)

	if cfg.ManageFeePercent <= 0 || cfg.ManageFeePercent > 100 {
		cfg.ManageFeePercent = defaultManageFeePercent
	}
	poolLog.Infof("Manage fee: %v%%", cfg.ManageFeePercent)

	// Warn about missing config file only after all other configuration is
	// done.  This prevents the warning on help messages and invalid
	// options.  Note this should go directly before the return.
	if configFileError != nil {
		poolLog.Warnf("%v", configFileError)
	}

	// Generate the TLS cert and key file if both don't already
	// exist.
	if !cfg.DisableTLS {
		if !fileExists(cfg.PoolCert) && !fileExists(cfg.PoolKey) {
			err := genCertPair(cfg.PoolCert, cfg.PoolKey, cfg.ExternalIPs)
			if err != nil {
				return nil, nil, err
			}
		}
	} else {
		poolLog.Infof("TLS certificate for websocket RPC server is disabled")
	}

	if cfg.EnableTCPSocket {
		if !cfg.DisableTCPSocketTLS {
			if !fileExists(cfg.PoolCert) && !fileExists(cfg.PoolKey) {
				err := genCertPair(cfg.PoolCert, cfg.PoolKey, cfg.ExternalIPs)
				if err != nil {
					return nil, nil, err
				}
			}
		} else {
			poolLog.Infof("TLS certificate for TCP socket server is disabled")
		}
	}

	if cfg.UseAddrFromAbec {
		poolLog.Infof("Use mining address provided by abec")
	} else {
		if len(cfg.MiningAddrs) == 0 {
			return nil, nil, errors.New("no mining address specified, please add miningaddr=addr1 miningaddr=addr2 into " +
				"configuration or add --useaddrfromabec option")
		}
		if len(cfg.MiningAddrs) == 1 {
			poolLog.Info("Find 1 mining address in total")
		} else {
			poolLog.Infof("Find %v mining addresses in total", len(cfg.MiningAddrs))
		}
		// Check address validity
		for i, addrStr := range cfg.MiningAddrs {
			abelAddr, err := utils.DecodeAbelAddress(addrStr)
			if err != nil {
				return nil, nil, fmt.Errorf("%d-th mining address failed to decode: %v", i, err)
			}
			if !abelAddr.IsForNet(chaincfg.ActiveNetParams) {
				return nil, nil, fmt.Errorf("%d-th mining address is on the wrong network", i)
			}
		}
	}

	if len(cfg.ServerManageFeeAddrs) != 0 {
		if len(cfg.ServerManageFeeAddrs) == 1 {
			poolLog.Info("Find 1 server management fee address in total")
		} else {
			poolLog.Infof("Find %v server management fee addresses in total", len(cfg.ServerManageFeeAddrs))
		}
		// Check address validity
		for i, addrStr := range cfg.ServerManageFeeAddrs {
			abelAddr, err := utils.DecodeAbelAddress(addrStr)
			if err != nil {
				return nil, nil, fmt.Errorf("%d-th server manangement fee address failed to decode: %v", i, err)
			}
			if !abelAddr.IsForNet(chaincfg.ActiveNetParams) {
				return nil, nil, fmt.Errorf("%d-th server management fee address is on the wrong network", i)
			}
		}
	} else {
		if cfg.UseWallet {
			return nil, nil, errors.New("No server management fee address specified, please specify at least one")
		}
	}

	// Validate profile port number
	if cfg.ProfilePort != "" {
		profilePort, err := strconv.Atoi(cfg.ProfilePort)
		if err != nil || profilePort < 1024 || profilePort > 65535 {
			str := "%s: The profile port must be between 1024 and 65535"
			err := fmt.Errorf(str, funcName)
			return nil, nil, err
		}
	}

	if len(cfg.AgentBlacklist) > 0 {
		poolLog.Infof("User-agent blacklist %s", cfg.AgentBlacklist)
	}
	if len(cfg.AgentWhitelist) > 0 {
		poolLog.Infof("User-agent whitelist %s", cfg.AgentWhitelist)
	}

	// Validate any given blacklisted IP addresses and networks.
	if len(cfg.Blacklists) > 0 {
		var ip net.IP
		cfg.blacklists = make([]*net.IPNet, 0, len(cfg.Blacklists))

		for _, addr := range cfg.Blacklists {
			_, ipnet, err := net.ParseCIDR(addr)
			if err != nil {
				ip = net.ParseIP(addr)
				if ip == nil {
					str := "%s: The blacklist value of '%s' is invalid"
					err = fmt.Errorf(str, funcName, addr)
					return nil, nil, err
				}
				var bits int
				if ip.To4() == nil {
					// IPv6
					bits = 128
				} else {
					bits = 32
				}
				ipnet = &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(bits, bits),
				}
			}
			cfg.blacklists = append(cfg.blacklists, ipnet)
		}
		poolLog.Infof("IP blacklist %s", cfg.Blacklists)
	}

	// Validate any given whitelisted IP addresses and networks.
	if len(cfg.Whitelists) > 0 {
		var ip net.IP
		cfg.whitelists = make([]*net.IPNet, 0, len(cfg.Whitelists))

		for _, addr := range cfg.Whitelists {
			_, ipnet, err := net.ParseCIDR(addr)
			if err != nil {
				ip = net.ParseIP(addr)
				if ip == nil {
					str := "%s: The whitelist value of '%s' is invalid"
					err = fmt.Errorf(str, funcName, addr)
					return nil, nil, err
				}
				var bits int
				if ip.To4() == nil {
					// IPv6
					bits = 128
				} else {
					bits = 32
				}
				ipnet = &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(bits, bits),
				}
			}
			cfg.whitelists = append(cfg.whitelists, ipnet)
		}
		poolLog.Infof("IP whitelist %s", cfg.Whitelists)
	}

	// Validate any given blacklisted IP addresses and networks.
	if len(cfg.AdminBlacklists) > 0 {
		var ip net.IP
		cfg.adminBlacklists = make([]*net.IPNet, 0, len(cfg.AdminBlacklists))

		for _, addr := range cfg.AdminBlacklists {
			_, ipnet, err := net.ParseCIDR(addr)
			if err != nil {
				ip = net.ParseIP(addr)
				if ip == nil {
					str := "%s: The admin blacklist value of '%s' is invalid"
					err = fmt.Errorf(str, funcName, addr)
					return nil, nil, err
				}
				var bits int
				if ip.To4() == nil {
					// IPv6
					bits = 128
				} else {
					bits = 32
				}
				ipnet = &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(bits, bits),
				}
			}
			cfg.adminBlacklists = append(cfg.adminBlacklists, ipnet)
		}
		poolLog.Infof("Admin IP blacklist %s", cfg.AdminBlacklists)
	}

	// Validate any given whitelisted IP addresses and networks.
	if len(cfg.AdminWhitelists) > 0 {
		var ip net.IP
		cfg.adminWhitelists = make([]*net.IPNet, 0, len(cfg.AdminWhitelists))

		for _, addr := range cfg.AdminWhitelists {
			_, ipnet, err := net.ParseCIDR(addr)
			if err != nil {
				ip = net.ParseIP(addr)
				if ip == nil {
					str := "%s: The admin whitelist value of '%s' is invalid"
					err = fmt.Errorf(str, funcName, addr)
					return nil, nil, err
				}
				var bits int
				if ip.To4() == nil {
					// IPv6
					bits = 128
				} else {
					bits = 32
				}
				ipnet = &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(bits, bits),
				}
			}
			cfg.adminWhitelists = append(cfg.adminWhitelists, ipnet)
		}
		poolLog.Infof("Admin IP whitelist %s", cfg.AdminWhitelists)
	}

	if cfg.ExtraNonce2BitLen <= 0 || cfg.ExtraNonce2BitLen > 64 {
		return nil, nil, fmt.Errorf("Invalid extra nonce 2 bit len: %v, should larger than 0 and smaller than or equal to 64", cfg.ExtraNonce2BitLen)
	}

	cfg.EthashConfig.VerifyByFullDAG = cfg.ethashVerifyByFullDAG
	chaincfg.PoolBackendVersion = version()

	return &cfg, remainingArgs, nil
}
