package sshConnect

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"code.google.com/marksheahan-sshblock/ssh"
	"code.google.com/p/go-uuid/uuid"
)

// Connection Define Connection
type Connection struct {
	User     string
	Host     string
	Password string
}

// SessionID Define Session ID
type SessionID struct {
	ID string
}

// Command Define Command Out
type Command struct {
	SessionID string
	Command   string
}

// Declare Maps
var clientList = make(map[string]*ssh.Client)
var sessionList = make(map[string]*ssh.Session)
var sessionIn = make(map[string]io.WriteCloser)
var sessionOut = make(map[string]io.Reader)
var sessionErr = make(map[string]io.Reader)

// SessionHandler get SSH session
func SessionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("GET : Not Implemented"))
	case "POST":
		body, _ := ioutil.ReadAll(r.Body)
		var c Connection
		json.Unmarshal(body, &c)
		uuid := uuid.New()
		client, session, errorSSH := connectToHost(c.User, c.Host, c.Password, uuid)
		if errorSSH != "OK" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{\"Status\" : \"" + errorSSH + "\"}"))
		} else {
			clientList[uuid] = client
			sessionList[uuid] = session
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{\"ID\" : \"" + uuid + "\", \"Status\" : \"OK\"}"))
		}
	case "PUT":
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("PUT : Not Implemented"))
	case "DELETE":
		body, _ := ioutil.ReadAll(r.Body)
		var s SessionID
		json.Unmarshal(body, &s)
		//closeSession
		closeSession(clientList[s.ID])
		delete(clientList, s.ID)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"ID\" : \"" + s.ID + "\", \"Status\" : \"Removed\"}"))
	}
}

// CommandHandler get command out
func CommandHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("GET : Not Implemented"))
	case "POST":
		body, _ := ioutil.ReadAll(r.Body)
		var c Command
		json.Unmarshal(body, &c)
		session := sessionList[c.SessionID]
		out := sendCommand(session, c.Command, c.SessionID)

		str := make([]string, len(bytes.Split([]byte(out), []byte{'\n'})))
		for i, line := range bytes.Split([]byte(out), []byte{'\n'}) {
			bs := make([]byte, 4)
			binary.LittleEndian.PutUint32(bs, 13)
			if len(line) != 0 {
				if bs[0] == line[len(line)-1] {
					if len(line)-1 != 0 {
						str[i] = string(line[:len(line)-1])
					}
				} else {
					str[i] = string(line)
				}
			} else {
			}
		}
		var commandOut string
		commandOut = "["
		for i, line := range str {
			commandOut += "\"" + line + "\""
			if i != len(str)-1 {
				commandOut += ","
			}
		}
		commandOut += "]"
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"CommandOut\" : " + commandOut + "}"))
	case "PUT":
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("PUT : Not Implemented"))
	case "DELETE":
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("DELETE : Not Implemented"))
	}
}

// connectToHost establish connection with Host
func connectToHost(user, host, password, uuid string) (client *ssh.Client, session *ssh.Session, errorSSH string) {
	// Create sshConfig variable
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.Password(password)},
		Config: ssh.Config{
			Ciphers: ssh.AllSupportedCiphers(),
		},
	}

	// Create Client
	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		errorSSH = err.Error()
	} else {
		// Create Session
		session, err = client.NewSession()
		if err != nil {
			client.Close()
			errorSSH = "Session_KO"
		} else {
			// Save Session
			sessionOut[uuid], err = session.StdoutPipe()
			sessionIn[uuid], _ = session.StdinPipe()
			sessionErr[uuid], _ = session.StderrPipe()
			session.Shell()
			var buf []byte
			sessionOut[uuid].Read(buf)
			errorSSH = "OK"
		}
	}
	// Return client, session and error
	return client, session, errorSSH

}

// sendCommand send command to Host
func sendCommand(session *ssh.Session, command, sessionID string) string {

	if sessionIn[sessionID] == nil {
		return "Session ID not exist"
	}

	buf := make([]byte, 1000)

	var loadStr bytes.Buffer

	// Send command
	switch command {
	case "":
		loadStr.WriteString("Command is Empty")
		return loadStr.String()
	case "\n":
		loadStr.WriteString("Command is not defined")
		return loadStr.String()
	default:
		sessionIn[sessionID].Write([]byte(command + "\n"))
		time.Sleep(1000 * time.Millisecond)

		n, _ := sessionOut[sessionID].Read(buf)
		var line string

		for n > 0 {
			if n < 1000 {
				break
			}
			line = strings.Replace(string(buf[:n]), "\"", "\\\"", -1)
			loadStr.WriteString(line)
			time.Sleep(1000 * time.Millisecond)
			n, _ = sessionOut[sessionID].Read(buf)
		}

		line = strings.Replace(string(buf[:n]), "\"", "\\\"", -1)
		loadStr.WriteString(line)

		return loadStr.String()
	}

}

// Close session
func closeSession(client *ssh.Client) {

	client.Close()

}
