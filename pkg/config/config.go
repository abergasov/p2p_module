package config

import "time"

type Config struct {
	P2P  P2P  `yaml:"p2p"`
	Http Http `yaml:"http"`
}

type Http struct {
	Port                         string `yaml:"port" envconfig:"HTTP_PORT"`
	Address                      string `yaml:"address" envconfig:"HTTP_ADDRESS"`
	MaxConcurrentStreams         uint32 `yaml:"max-concurrent-streams" envconfig:"HTTP_MAX_CONCURRENT_STREAMS"`
	MaxUploadBufferPerStream     int32  `yaml:"max-stream-size" envconfig:"HTTP_MAX_STREAM_SIZE"`
	MaxReadFrameSize             uint32 `yaml:"max-frame-size" envconfig:"HTTP_MAX_FRAME_SIZE"`
	MaxUploadBufferPerConnection int32  `yaml:"initial-window-size" envconfig:"HTTP_INITIAL_WINDOW_SIZE"`
}

type P2P struct {
	GracePeersShutdown time.Duration `yaml:"grace-peers-shutdown" envconfig:"P2P_GRACE_PEERS_SHUTDOWN"`
	DataDir            string        `yaml:"data-dir" envconfig:"P2P_DATA_DIR"`
	BootNode           string        `yaml:"boot_node" envconfig:"P2P_BOOT_NODE"`
	LowPeers           int           `yaml:"low-peers" envconfig:"P2P_LOW_PEERS"`
	HighPeers          int           `yaml:"high-peers" envconfig:"P2P_HIGH_PEERS"`
	Listen             string        `yaml:"listen" envconfig:"P2P_LISTEN"`
}
