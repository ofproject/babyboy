package node

import (
	"babyboy-dag/p2p"
	"babyboy-dag/p2p/nat"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// DefaultConfig contains reasonable default settings.
var DefaultConfig = Config{
	DataDir:      "data/",
	RpcServer:    "0.0.0.0:8545",
	RemoteServer: "http://192.168.1.13:8888",
	P2P: p2p.Config{
		ListenAddr: ":3000",
		MaxPeers:   25,
		NAT:        nat.Any(),
	},
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "BabyBoy")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "BabyBoy")
		} else {
			return filepath.Join(home, ".BabyBoy")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
