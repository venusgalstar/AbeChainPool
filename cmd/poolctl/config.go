package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/utils"
	"github.com/jessevdk/go-flags"
)

const (
	// unusableFlags are the command usage flags which this utility are not
	// able to use. In particular it doesn't support websockets and
	// consequently notifications.
	unusableFlags = pooljson.UFWebsocketOnly | pooljson.UFNotification
)

var (
	poolServerHomeDir  = utils.AppDataDir("abe-miningpool-server", false)
	poolctlHomeDir     = utils.AppDataDir("abe-poolctl", false)
	defaultConfigFile  = "poolctl.conf"
	defaultRPCServer   = "localhost"
	defaultRPCCertFile = "pool.cert"
)

// config defines the configuration options for poolctl.
//
// See loadConfig for details on the configuration load process.
type config struct {
	ConfigFile  string `short:"C" long:"configfile" description:"Path to configuration file"`
	NoTLS       bool   `long:"notls" description:"Disable TLS"`
	Proxy       string `long:"proxy" description:"Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	ProxyPass   string `long:"proxypass" default-mask:"-" description:"Password for proxy server"`
	ProxyUser   string `long:"proxyuser" description:"Username for proxy server"`
	PoolAddress string `short:"s" long:"pooladdress" description:"RPC server to connect to"`
	PoolCert    string `short:"c" long:"poolcert" description:"RPC server certificate chain for validation"`
	RPCPassword string `short:"P" long:"rpcpass" default-mask:"-" description:"RPC password"`
	RPCUser     string `short:"u" long:"rpcuser" description:"RPC username"`
	ShowVersion bool   `short:"V" long:"version" description:"Display version information and exit"`
	WorkingDir  string `long:"workingdir" description:"Working directory"`
}

// createDefaultConfig creates a basic config file at the given destination path.
// For this it tries to read the config file for the RPC server (either pool server or
// pool client), and extract the RPC user and password from it.
func createDefaultConfigFile(destinationPath, serverConfigPath string) error {
	// Read the RPC server config
	serverConfigFile, err := os.Open(serverConfigPath)
	if err != nil {
		return err
	}
	defer serverConfigFile.Close()
	content, err := ioutil.ReadAll(serverConfigFile)
	if err != nil {
		return err
	}

	// Extract the rpcuser
	rpcUserRegexp := regexp.MustCompile(`(?m)^\s*rpcuser=([^\s]+)`)
	userSubmatches := rpcUserRegexp.FindSubmatch(content)
	if userSubmatches == nil {
		// No user found, nothing to do
		return nil
	}

	// Extract the rpcpass
	rpcPassRegexp := regexp.MustCompile(`(?m)^\s*rpcpass=([^\s]+)`)
	passSubmatches := rpcPassRegexp.FindSubmatch(content)
	if passSubmatches == nil {
		// No password found, nothing to do
		return nil
	}

	// Create the destination directory if it does not exists
	err = os.MkdirAll(filepath.Dir(destinationPath), 0700)
	if err != nil {
		return err
	}

	// Create the destination file and write the rpcuser and rpcpass to it
	dest, err := os.OpenFile(destinationPath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dest.Close()

	destString := fmt.Sprintf("rpcuser=%s\nrpcpass=%s\n",
		string(userSubmatches[1]), string(passSubmatches[1]))

	dest.WriteString(destString)

	return nil
}

// cleanAndExpandPath expands environement variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(poolctlHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but they variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

// normalizeAddress returns addr with the passed default port appended if
// there is not already a port specified.
func normalizeAddress(addr string) (string, error) {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		defaultPort := "27777"

		return net.JoinHostPort(addr, defaultPort), nil
	}
	return addr, nil
}

func loadConfig() (*config, []string, error) {
	// Default config.
	cfg := config{
		ConfigFile:  defaultConfigFile,
		PoolAddress: defaultRPCServer,
		PoolCert:    defaultRPCCertFile,
	}

	// Pre-parse the command line options to see if an alternative config
	// file, the version flag, or the list commands flag was specified.  Any
	// errors aside from the help message error can be ignored here since
	// they will be caught by the final parse below.
	preCfg := cfg
	preParser := flags.NewParser(&preCfg, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "The special parameter `-` "+
				"indicates that a parameter should be read "+
				"from the\nnext unread line from standard "+
				"input.")
			return nil, nil, err
		}
	}

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	usageMessage := fmt.Sprintf("Use %s -h to show options", appName)
	if preCfg.ShowVersion {
		fmt.Println(appName, "version", version())
		os.Exit(0)
	}

	if preCfg.WorkingDir != "" {
		err := os.Chdir(preCfg.WorkingDir)
		if err != nil {
			return nil, nil, err
		}
	}

	// Load additional config from file.
	parser := flags.NewParser(&cfg, flags.Default)
	err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config file: %v\n",
				err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, nil, err
		}
	}

	// Parse command line options again to ensure they take precedence.
	remainingArgs, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			fmt.Fprintln(os.Stderr, usageMessage)
		}
		return nil, nil, err
	}

	// Handle environment variable expansion in the RPC certificate path.
	cfg.PoolCert = cleanAndExpandPath(cfg.PoolCert)

	// Add default port to RPC server based on --client
	cfg.PoolAddress, err = normalizeAddress(cfg.PoolAddress)
	if err != nil {
		return nil, nil, err
	}

	return &cfg, remainingArgs, nil
}
