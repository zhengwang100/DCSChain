package factory

import (
	"bcrequest"
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"message"
	"os"
	"p2p"
	"server"
	"ssm2"
	"strconv"
)

// GenNewReq: generate new request
// note: if the view leader is in new-view phase, send the request to the leader, or send it to next view leader
// params:
// simulateServers: the slice of nodes in system
// input:			the new req
// func GenNewReq(simulateServers []*server.Server, reqs [][]byte, signs [][]byte) {
func GenNewReq(simulateServers []*server.Server, reqs []bcrequest.BCRequest) {

	// when the input is empty, generate random requests
	if len(reqs) == 0 {
		// input := common.GenerateSecureRandomStringSlice(1, 10)
		// reqs = common.StringSlice2TwoDimByteSlice(input)
		return
	}

	for {
		for i, s := range simulateServers {
			if s.Orderer.IsLeader() {
				// fmt.Println(s.ServerID.ID.Name, s.Orderer.PBFTConsensus.View, s.Orderer.IsWaitingReq(), len(s.Requests)+len(reqs) <= s.BatchSize)
				// check whether the leader is waiting request
				if s.RequestsLock.TryLock() {
					if s.Orderer.IsWaitingReq() {
						if len(s.Requests)+len(reqs) <= s.BatchSize {
							// s.Logger.Println("push leader", s.ServerID.ID.Name, s.Orderer.PBFTConsensus.CurPhase, s.Orderer.PBFTConsensus.View.ViewNumber)
							// 	s.ServerID.ID.Name,
							// 	s.Orderer.PBFTConsensus.View.ViewNumber,
							// 	s.Orderer.PBFTConsensus.View.Leader,
							// 	s.Orderer.IsWaitingReq(),
							// 	reqs[0].Cmd[:5])

							simulateServers[i].Requests = append(simulateServers[i].Requests, reqs...)
							s.RequestsLock.Unlock()

							// send the flag of handle request
							simulateServers[i].Orderer.ReqFlagChan <- true
							return
						}
					}
					s.RequestsLock.Unlock()
				}
			}

		}
		// time.Sleep(40 * time.Millisecond)
	}
}

// AutoGenChainedNewReq: auto-generate requests
// params:
// - simulateServers: the slice of nodes in system
// - params[]:
// -- 1.count: Number of requests
// -- 2.reqNum: The number of transactions per request
// -- 3.length: Length of each request
func AutoGenNewReq(simulateServers []*server.Server, params []string) {
	// sk := ssm2.ReadKey("../../client/config/public.pem")

	// analytic three parameters, default parameters are (1,1,10)
	paramInt := make([]int, 3)
	if len(params) < 3 {
		paramInt[0], paramInt[1], paramInt[2] = 1, 128, 128
	} else {
		for i := range paramInt {
			val, err := strconv.Atoi(params[i])
			if err == nil {
				paramInt[i] = val
			} else {
				paramInt[i] = i
			}
		}
	}

	for _, s := range simulateServers {
		s.BatchSize = paramInt[1]
	}

	count := 0
	// reqs := ReadReq(paramInt[1]*paramInt[0], paramInt[2])
	reqs := ReadReq(paramInt[1], paramInt[2])
	startMsg := &message.ServerMsg{SendServer: "start", Payload: []byte{byte(len(simulateServers))}}
	msgJson, _ := json.Marshal(startMsg)
	conn := p2p.Send(simulateServers[0].Clients["c_0"].Conn, simulateServers[0].Clients["c_0"].Addr, append(msgJson, []byte("\n")...))
	if conn != nil {
		simulateServers[0].Clients["c_0"].Conn = conn
	}

	// generate chained request according to the parameters
	for i := 0; i < paramInt[0]; i++ {
		fmt.Println(i)
		// reqs[0].Cmd[0] = byte(i)
		GenNewReq(simulateServers, reqs)
		// time.Sleep(30 * time.Millisecond)
		count += paramInt[1]
		// reqs = ReadReq(paramInt[1], paramInt[2])
	}
	fmt.Println("Finish auto generate request number:", count)
}

// ReadReq: read request from file
// params:
// - count:  the count of the request want to read
// - length: the length of the request want to read
func ReadReq(count int, length int) []bcrequest.BCRequest {
	reqs := make([]bcrequest.BCRequest, count)
	file, err := os.Open("../../config/request/request" + strconv.Itoa(length))
	if err != nil {
		file, err = os.Open("./config/request/request" + strconv.Itoa(length))
		if err != nil {
			log.Fatalf("Failed to open file: %v", err)
		}
	}
	defer file.Close()

	// 创建 bufio.Scanner
	scanner := bufio.NewScanner(file)

	for i := 0; i < count; i++ {

		// double if clause guarantees that two lines can be read at once
		if scanner.Scan() {
			cmd := []byte{}
			json.Unmarshal(scanner.Bytes(), &cmd)
			if scanner.Scan() {
				sign := []byte{}
				json.Unmarshal(scanner.Bytes(), &sign)
				reqs[i] = bcrequest.BCRequest{
					Id:   "c_0",
					Cmd:  cmd,
					Sign: sign,
				}
				// reqs[i] = req
				// signs[i] = sign

			} else {
				log.Fatalf("Failed to read signature to file: %v", err)
				i--
				continue
			}
		} else {
			log.Fatalf("Failed to read requests to file: %v", err)
			i--
			continue
		}
	}

	return reqs
}

// SignCmd: generate the signature for the command
// params:
// - cmds: the command to be signed
func SignCmd(cmds [][]byte) []bcrequest.BCRequest {
	signer := ssm2.Signer{}
	signer.Sk = signer.GetSKFromFile()
	signer.Pk = signer.GetPKFromFile()
	count := len(cmds)
	reqs := make([]bcrequest.BCRequest, len(cmds))
	for i := 0; i < count; i++ {
		reqs[i] = bcrequest.BCRequest{
			Id:   "c_0",
			Cmd:  cmds[i],
			Sign: signer.Sign(cmds[i]),
		}
	}

	return reqs
}
