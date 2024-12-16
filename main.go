package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-routeros/routeros/v3"
	"github.com/joho/godotenv"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type LPACRequest struct {
	Type    string  `json:"type"`
	Payload Request `json:"payload"`
}

type Request struct {
	Func  string      `json:"func"`
	Param interface{} `json:"param"`
}

type LPACResponse struct {
	Type    string   `json:"type"`
	Payload Response `json:"payload"`
}

type Response struct {
	ECode *int        `json:"ecode,omitempty"`
	Data  interface{} `json:"data,omitempty"`
	Env   string      `json:"env,omitempty"`
}

type Interface struct {
	Id   string `json:"env"`
	Bus  string `json:"-"`
	Name string `json:"name"`
}

var rosClient *routeros.Client
var cchoRegex = regexp.MustCompile("\\+CCHO:\\s?(\\d+)")
var cglaRegex = regexp.MustCompile("\\+CGLA:\\s?(\\d+),(\\S+)")

var sessionId = -1

func sendATCommand(cmd string) string {
	return sendATCommandToDev(cmd, os.Getenv("DEVICE_IFID"))
}
func sendATCommandToDev(cmd string, dev string) string {
	r, err := rosClient.Run("/interface/lte/at-chat", "=.id="+dev, "=input="+cmd)
	if err != nil {
		return ""
	}
	if len(r.Re) == 0 {
		panic("failed to communicate with modem, " + r.String())
	}

	if len(r.Re[0].List) == 0 {
		panic("failed to communicate with modem, " + r.Re[0].String())
	}
	return fmt.Sprintf(r.Re[0].List[0].Value)
}

func getLTEInterface() []Interface {
	r, err := rosClient.Run("/interface/lte/print")
	if err != nil {
		panic("failed to execute command on RouterOS, " + err.Error())
	}
	if len(r.Re) == 0 {
		return nil
	}

	var ifs []Interface

	for _, intf := range r.Re {
		if intf.Map["running"] == "true" {
			caps, err := rosClient.Run("/interface/lte/show-capabilities", "=.id="+intf.Map["name"])
			if err != nil {
				panic("failed to execute command on RouterOS, " + err.Error())
			}

			if at, exists := caps.Re[0].Map["at-chat"]; exists && at == "true" {
				if sendATCommandToDev("AT", intf.Map["name"]) == "OK" {
					ifs = append(ifs,
						Interface{
							Id:   intf.Map["name"],
							Bus:  caps.Re[0].Map["modem-bus-location"],
							Name: fmt.Sprintf("%s at %s", intf.Map["name"], caps.Re[0].Map["modem-bus-location"]),
						})
				}

			}
		}
	}

	return ifs
}

func main() {
	var err error
	if _, err = os.Stat(".env"); err == nil {
		err = godotenv.Load(".env")
		if err != nil {
			panic("Error loading.env file, " + err.Error())
		}
	}

	if len(os.Args) == 1 {
		fmt.Println("Usage: lpac <original lpac command>")
		os.Exit(1)
	}

	lpacBin := "./lpac.orig"
	fileName := filepath.Base(os.Args[0])

	if runtime.GOOS == "windows" {
		if fileName == "lpac.exe" {
			if _, err = os.Stat("./lpac.orig.exe"); errors.Is(err, os.ErrNotExist) {
				panic("Please copy the original lpac.exe to lpac.orig.exe")
			}
			lpacBin = "./lpac.orig.exe"
		} else {
			if _, err = os.Stat("./lpac.exe"); errors.Is(err, os.ErrNotExist) {
				panic("Please download lpac from Github and put it with this program")
			}
			lpacBin = "./lpac.exe"
		}
	} else {
		if fileName == "lpac" {
			if _, err = os.Stat("./lpac.orig"); errors.Is(err, os.ErrNotExist) {
				panic("Please copy the original lpac to lpac.orig")
			}
		} else {
			path := ""
			path, err = exec.LookPath("lpac")
			if err == nil {
				lpacBin = path
			} else {
				if _, err = os.Stat("./lpac"); errors.Is(err, os.ErrNotExist) {
					panic("Please download lpac from Github and put it with this program")
				}
				lpacBin = "./lpac"
			}
		}
	}

	rosClient, err = routeros.Dial(
		fmt.Sprintf("%s:%s",
			os.Getenv("ROS_IP"),
			os.Getenv("ROS_API_PORT")),
		os.Getenv("ROS_LOGIN"),
		os.Getenv("ROS_PASSWORD"))
	if err != nil {
		panic("Failed to connect to RouterOS API, " + err.Error())
	}

	interfaces := getLTEInterface()
	if len(interfaces) == 0 {
		panic("No interfaces available for at-chat in RouterOS")
	}

	if os.Getenv("DEVICE_IFID") == "" {
		_ = os.Setenv("DEVICE_IFID", interfaces[0].Id)
	}

	if strings.ToLower(os.Args[1]) == "driver" &&
		strings.ToLower(os.Args[2]) == "apdu" &&
		strings.ToLower(os.Args[3]) == "list" {

		resp := LPACResponse{
			Type: "lpa",
			Payload: Response{
				Env: fmt.Sprintf("%s:%s",
					os.Getenv("ROS_IP"),
					os.Getenv("ROS_API_PORT")),
				Data: interfaces,
			},
		}
		respJson, _ := json.Marshal(resp)
		_, _ = os.Stdout.WriteString(string(respJson))
		os.Exit(0)
	}

	lpacCmd := exec.Command(lpacBin, os.Args[1:]...)
	lpacCmd.Env = append(lpacCmd.Env, "LPAC_APDU=stdio")

	stdout, _ := lpacCmd.StdoutPipe()
	stdin, _ := lpacCmd.StdinPipe()

	stdinReader := bufio.NewReader(stdout)
	stdoutWriter := bufio.NewWriter(stdin)
	if err = lpacCmd.Start(); err != nil {
		panic("Failed to launch lpac, " + err.Error())
	}

	go func() {
		for {

			input, _ := stdinReader.ReadString('\n')

			if input == "" {
				continue
			}

			var req LPACRequest
			if err = json.Unmarshal([]byte(input), &req); err != nil {
				fmt.Print(input)
				continue
			}

			if req.Type != "apdu" {
				fmt.Print(input)
				continue
			}

			resp := LPACResponse{Type: "apdu", Payload: Response{ECode: new(int)}}

			switch req.Payload.Func {
			case "connect":
				*resp.Payload.ECode = 0
				sendATCommand("AT+CCHC=1")
				sendATCommand("AT+CCHC=2")
				sendATCommand("AT+CCHC=3")
				sendATCommand("AT+CCHC=4")
			case "disconnect":
				*resp.Payload.ECode = 0
			case "logic_channel_close":
				sendATCommand("AT+CCHC=" + req.Payload.Param.(string))
			case "transmit":
				apdu := req.Payload.Param.(string)
				atResp := sendATCommand(fmt.Sprintf("AT+CGLA=%d,%d,\"%s\"", sessionId, len(apdu), apdu))
				cglaResp := cglaRegex.FindAllStringSubmatch(atResp, -1)
				*resp.Payload.ECode = sessionId
				resp.Payload.Data = strings.Replace(cglaResp[0][2], "\"", "", -1)

			case "logic_channel_open":
				atResp := sendATCommand("AT+CCHO=\"" + req.Payload.Param.(string) + "\"")
				if strings.LastIndex(atResp, "ERROR") > 0 {
					panic("failed to open logic sessionId, resp=" + atResp)
				}
				cchoResp := cchoRegex.FindAllStringSubmatch(atResp, -1)
				sessionId, _ = strconv.Atoi(cchoResp[0][1])
				*resp.Payload.ECode = sessionId
			}

			respJson, _ := json.Marshal(resp)
			_, _ = stdoutWriter.WriteString(string(respJson) + "\r\n")
			_ = stdoutWriter.Flush()

		}
	}()

	_ = lpacCmd.Wait()
	_ = rosClient.Close()
}
