// Copyright (c) 2022 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package ssh

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/blacknon/go-sshlib"
	"github.com/blacknon/lssh/common"
	"golang.org/x/crypto/ssh"
)

// run shell
func (r *Run) shell() (err error) {
	// server config
	server := r.ServerList[0]
	config := r.Conf.Server[server]

	// check count AuthMethod
	if len(r.serverAuthMethodMap[server]) == 0 {
		msg := fmt.Sprintf("Error: %s is No AuthMethod.\n", server)
		err = errors.New(msg)
		return
	}

	// set port forwarding
	config = r.setPortForwards(server, config)

	// OverWrite dynamic port forwarding
	if r.DynamicPortForward != "" {
		config.DynamicPortForward = r.DynamicPortForward
	}

	// OverWrite reverse dynamic port forwarding
	if r.ReverseDynamicPortForward != "" {
		config.ReverseDynamicPortForward = r.ReverseDynamicPortForward
	}

	// OverWrite http dynamic port forwarding
	if r.HTTPDynamicPortForward != "" {
		config.HTTPDynamicPortForward = r.HTTPDynamicPortForward
	}

	// OverWrite http reverse dynamic port forwarding
	if r.HTTPReverseDynamicPortForward != "" {
		config.HTTPReverseDynamicPortForward = r.HTTPReverseDynamicPortForward
	}

	// OverWrite nfs dynacmic forwarding
	if r.NFSDynamicForwardPort != "" {
		config.NFSDynamicForwardPort = r.NFSDynamicForwardPort
	}

	// OverWrite nfs dynamic path
	if r.NFSDynamicForwardPath != "" {
		config.NFSDynamicForwardPath = r.NFSDynamicForwardPath
	}

	// OverWrite nfs reverse dynamic forwarding
	if r.NFSReverseDynamicForwardPort != "" {
		config.NFSReverseDynamicForwardPort = r.NFSReverseDynamicForwardPort
	}

	// OverWrite nfs reverse dynamic path
	if r.NFSReverseDynamicForwardPath != "" {
		config.NFSReverseDynamicForwardPath = r.NFSReverseDynamicForwardPath
	}

	// OverWrite local bashrc use
	if r.IsBashrc {
		config.LocalRcUse = "yes"
	}

	// OverWrite local bashrc not use
	if r.IsNotBashrc {
		config.LocalRcUse = "no"
	}

	// header
	r.PrintSelectServer()
	for _, fw := range config.Forwards {
		r.printPortForward(fw.Mode, fw.Local, fw.Remote)
	}
	r.printDynamicPortForward(config.DynamicPortForward)
	r.printReverseDynamicPortForward(config.ReverseDynamicPortForward)
	r.printHTTPDynamicPortForward(config.HTTPDynamicPortForward)
	r.printProxy(server)
	if config.LocalRcUse == "yes" {
		fmt.Fprintf(os.Stderr, "Information   :This connect use local bashrc.\n")
	}

	// Craete sshlib.Connect (Connect Proxy loop)
	connect, err := r.CreateSshConnect(server)
	if err != nil {
		return
	}

	// Create session
	session, err := connect.CreateSession()
	if err != nil {
		return
	}

	// ssh-agent
	if config.SSHAgentUse {
		connect.Agent = r.agent
		connect.ForwardSshAgent(session)
	}

	// Local/Remote Port Forwarding
	for _, fw := range config.Forwards {
		// port forwarding
		switch fw.Mode {
		case "L", "":
			err = connect.TCPLocalForward(fw.Local, fw.Remote)
		case "R":
			err = connect.TCPRemoteForward(fw.Local, fw.Remote)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	// Dynamic Port Forwarding
	if config.DynamicPortForward != "" {
		go connect.TCPDynamicForward("localhost", config.DynamicPortForward)
	}

	// Reverse Dynamic Port Forwarding
	if config.ReverseDynamicPortForward != "" {
		go connect.TCPReverseDynamicForward("localhost", config.ReverseDynamicPortForward)
	}

	// HTTP Dynamic Port Forwarding
	if config.HTTPDynamicPortForward != "" {
		go connect.HTTPDynamicForward("localhost", config.HTTPDynamicPortForward)
	}

	// HTTP Reverse Dynamic Port Forwarding
	if config.HTTPReverseDynamicPortForward != "" {
		go connect.HTTPReverseDynamicForward("localhost", config.HTTPReverseDynamicPortForward)
	}

	// NFS Dynamic Forwarding
	if config.NFSDynamicForwardPort != "" && config.NFSDynamicForwardPath != "" {
		go connect.NFSForward("localhost", config.NFSDynamicForwardPort, config.NFSDynamicForwardPath)
	}

	// NFS Reverse Dynamic Forwarding
	if config.NFSReverseDynamicForwardPort != "" && config.NFSReverseDynamicForwardPath != "" {
		go connect.NFSReverseForward("localhost", config.NFSReverseDynamicForwardPort, config.NFSReverseDynamicForwardPath)
	}

	// switch check Not-execute flag
	// TODO(blacknon): Backgroundフラグを実装したら追加
	switch {
	case r.IsNone:
		r.noneExecute(connect)

	default:
		// run pre local command
		if config.PreCmd != "" {
			execLocalCommand(config.PreCmd)
		}

		// defer run post local command
		if config.PostCmd != "" {
			defer execLocalCommand(config.PostCmd)
		}

		// if terminal log enable
		logConf := r.Conf.Log
		if logConf.Enable {
			logPath := r.getLogPath(server)

			// Check logging with remove ANSI code flag.
			if logConf.RemoveAnsiCode {
				connect.SetLogWithRemoveAnsiCode(logPath, logConf.Timestamp)
			} else {
				connect.SetLog(logPath, logConf.Timestamp)
			}
		}

		// TODO(blacknon): local rc file add
		if config.LocalRcUse == "yes" {
			err = localrcShell(connect, session, config.LocalRcPath, config.LocalRcDecodeCmd, config.LocalRcCompress, config.LocalRcUncompressCmd)
		} else {
			// Connect shell
			err = connect.Shell(session)
		}
	}

	return
}

// getLogPath return log file path.
func (r *Run) getLogPath(server string) (logPath string) {
	// check regex
	// if ~/.ssh/config, in ":"
	reg := regexp.MustCompile(`:`)

	if reg.MatchString(server) {
		slice := strings.SplitN(server, ":", 2)
		server = slice[1]
	}

	dir, err := r.getLogDirPath(server)
	if err != nil {
		log.Println(err)
	}

	file := time.Now().Format("20060102_150405") + "_" + server + ".log"
	logPath = dir + "/" + file

	return
}

// getLogDirPath return log directory path
func (r *Run) getLogDirPath(server string) (dir string, err error) {
	u, _ := user.Current()
	logConf := r.Conf.Log

	// expantion variable
	dir = logConf.Dir
	dir = strings.Replace(dir, "~", u.HomeDir, 1)
	dir = strings.Replace(dir, "<Date>", time.Now().Format("20060102"), 1)
	dir = strings.Replace(dir, "<Hostname>", server, 1)

	// create directory
	err = os.MkdirAll(dir, 0700)

	return
}

// noneExecute is not execute command and shell.
func (r *Run) noneExecute(con *sshlib.Connect) (err error) {
loop:
	for {
		select {
		case <-time.After(500 * time.Millisecond):
			// 接続状況チェック
			err = con.CheckClientAlive()
			if err != nil {
				// error
				fmt.Fprintf(os.Stderr, "Exit Connect, Error: %s\n", err)

				// close sftp client
				con.Client.Close()

				break loop
			}

			continue loop
		}
	}

	return
}

// localRcShell connect to remote shell using local bashrc
func localrcShell(connect *sshlib.Connect, session *ssh.Session, localrcPath []string, decoder string, compress bool, uncompress string) (err error) {
	// var
	var cmd string

	// TODO(blacknon): 受け付けるrcdataをzip化するオプションの追加

	// set default bashrc
	if len(localrcPath) == 0 {
		localrcPath = []string{"~/.bashrc"}
	}

	// get bashrc base64 data
	// rcData, err := common.GetFilesBase64(localrcPath, common.ARCHIVE_NONE)
	rcData, _ := common.GetFilesBase64(localrcPath, common.ARCHIVE_GZIP)

	// set default uncompress command
	if uncompress == "" {
		uncompress = "gzip -d"
	}

	// switch
	switch {
	case !compress && decoder != "":
		cmd = fmt.Sprintf("bash --noprofile --rcfile <(echo %s | %s)", rcData, decoder)

	case !compress && decoder == "":
		cmd = fmt.Sprintf("bash --noprofile --rcfile <(echo %s | ( (base64 --help | grep -q coreutils) && base64 -d <(cat) || base64 -D <(cat) ) )", rcData)

	case compress && decoder != "":
		cmd = fmt.Sprintf("bash --noprofile --rcfile <(echo %s | %s | %s)", rcData, decoder, uncompress)

	case compress && decoder == "":
		cmd = fmt.Sprintf("bash --noprofile --rcfile <(echo %s | ( (base64 --help | grep -q coreutils) && base64 -d <(cat) || base64 -D <(cat) ) | %s)", rcData, uncompress)

	}

	connect.CmdShell(session, cmd)

	return
}
