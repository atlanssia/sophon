package conf

import (
	"encoding/json"
	"os"
)

type Option struct {
	Hostname        string `json:"hostname"`
	StartTLS        bool   `json:"start_tls"`
	ListenInterface string `json:"listen_interface"`
	PrivateKeyFile  string `json:"private_key_file"`
	PublicKeyFile   string `json:"public_key_file"`
	MaxSessions     int    `json:"max_sessions"`
	MaxMessageSize  int64  `json:"max_message_size"`
	LogFile         string `json:"log_file"`
	Welcoming       string `json:"welcoming"`
}

// load conf
func Load() (*Option, error) {

	// load conf
	file, err := os.Open("conf/conf.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	option := &Option{}
	err = decoder.Decode(option)
	// TODO check all fields valid
	// TODO check not empty, valid interface, valid key file, valid log file path etc.
	return option, err
}
