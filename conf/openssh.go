// Copyright (c) 2022 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package conf

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/blacknon/lssh/common"
	"github.com/kevinburke/ssh_config"
)

// readOpenSSHConfig open the OpenSSH configuration file, return *ssh_config.Config.
func readOpenSSHConfig(path, command string) (cfg *ssh_config.Config, err error) {
	var rd io.Reader
	switch {
	case path != "": // 1st
		// Read OpenSSH Config
		sshConfigFile := common.GetFullPath(path)
		rd, err = os.Open(sshConfigFile)
	case command != "": // 2nd
		var data []byte
		cmd := exec.Command("sh", "-c", command)
		data, err = cmd.Output()
		rd = bytes.NewReader(data)
	}

	// error check
	if err != nil {
		return
	}

	cfg, err = ssh_config.Decode(rd)
	return
}

// getOpenSSHConfig loads the specified OpenSSH configuration file and returns it in conf.ServerConfig format
func getOpenSSHConfig(path, command string) (config map[string]ServerConfig, err error) {
	config = map[string]ServerConfig{}

	// open openSSH config
	cfg, err := readOpenSSHConfig(path, command)
	if err != nil {
		return
	}

	// set name element
	ele := path
	if ele == "" {
		ele = "generate_sshconfig"
	}

	// Get Node names
	hostList := []string{}
	for _, h := range cfg.Hosts {
		// not supported wildcard host
		re := regexp.MustCompile("\\*")
		for _, pattern := range h.Patterns {
			if !re.MatchString(pattern.String()) {
				hostList = append(hostList, pattern.String())
			}
		}
	}

	// append ServerConfig
	for _, host := range hostList {
		serverConfig := ServerConfig{
			Addr:         ssh_config.Get(host, "HostName"),
			Port:         ssh_config.Get(host, "Port"),
			User:         ssh_config.Get(host, "User"),
			ProxyCommand: ssh_config.Get(host, "ProxyCommand"),
			PreCmd:       ssh_config.Get(host, "LocalCommand"),
			Note:         "from:" + ele,
		}

		if serverConfig.Addr == "" {
			serverConfig.Addr = host
		}

		if serverConfig.User == "" {
			serverConfig.User = os.Getenv("USER")
		}

		// TODO(blacknon): OpenSSSH設定ファイルだと、Certificateは複数指定可能な模様。ただ、あまり一般的な使い方ではないようなので、現状は複数のファイルを受け付けるように作っていない。
		key := ssh_config.Get(host, "IdentityFile")
		cert := ssh_config.Get(host, "Certificate")
		if cert != "" {
			serverConfig.Cert = cert
			serverConfig.CertKey = key
		} else {
			serverConfig.Key = key
		}

		// PKCS11 provider
		pkcs11Provider := ssh_config.Get(host, "PKCS11Provider")
		if pkcs11Provider != "" {
			serverConfig.PKCS11Use = true
			serverConfig.PKCS11Provider = pkcs11Provider
		}

		// x11 forwarding
		x11 := ssh_config.Get(host, "ForwardX11")
		if x11 == "yes" {
			serverConfig.X11 = true
		}

		// Port forwarding (Local forward)
		localForward := ssh_config.Get(host, "LocalForward")
		if localForward != "" {
			array := strings.SplitN(localForward, " ", 2)
			if len(array) > 1 {
				var e error

				_, e = strconv.Atoi(array[0])
				if e != nil { // localhost:8080
					serverConfig.PortForwardLocal = array[0]
				} else { // 8080
					serverConfig.PortForwardLocal = "localhost:" + array[0]
				}

				_, e = strconv.Atoi(array[1])
				if e != nil { // localhost:8080
					serverConfig.PortForwardRemote = array[1]
				} else { // 8080
					serverConfig.PortForwardRemote = "localhost:" + array[1]
				}
			}
		}

		// Port forwarding (Remote forward)
		remoteForward := ssh_config.Get(host, "RemoteForward")
		if remoteForward != "" {
			array := strings.SplitN(remoteForward, " ", 2)
			if len(array) > 1 {
				var e error

				_, e = strconv.Atoi(array[0])
				if e != nil { // localhost:8080
					serverConfig.PortForwardLocal = array[0]
				} else { // 8080
					serverConfig.PortForwardLocal = "localhost:" + array[0]
				}

				_, e = strconv.Atoi(array[1])
				if e != nil { // localhost:8080
					serverConfig.PortForwardRemote = array[1]
				} else { // 8080
					serverConfig.PortForwardRemote = "localhost:" + array[1]
				}
			}
		}

		// Port forwarding (Dynamic forward)
		dynamicForward := ssh_config.Get(host, "DynamicForward")
		if dynamicForward != "" {
			serverConfig.DynamicPortForward = dynamicForward
		}

		serverName := ele + ":" + host
		config[serverName] = serverConfig
	}

	return config, err
}
