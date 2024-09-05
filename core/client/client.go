package client

import (
	"bufio"
	"crypto/rand"
	"deltachain/common/dcs"
	"encoding/json"
	"fmt"
	hstypes "hotstuff/types"
	"identity"
	"io"
	"log"
	"message"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xlcetc/cryptogm/sm/sm2"
	"github.com/xuri/excelize/v2"
)

// Client:
type Client struct {
	ID            identity.PrivID       // ID represents the unique identifier for the client
	Address       string                // Address specifies the address of the client
	Port          string                // Port specifies the port number used by the client
	SerAddr       string                // server address, which may be used for specific server communications
	ServerAddr    map[int]string        // a map that stores the addresses of different servers
	ReplyMessages map[int][]interface{} // a map that stores the reply messages for different requests
	Logger        log.Logger            `json:"logger"` // logger responsible for logging
	Mu            sync.Mutex

	// the variable for test
	startTime  int64 // the timestamp when the test starts
	endTime    int64 // the timestamp when the test ends
	reqNum     int   // the number of requests sent during the test
	batchSize  int   // the Bathchsize of requests in the test
	viewCount  int   // the number of views the client has accessed
	startView  int   // the view number at the start of the test
	endView    int   // the view number at the end of the test
	showedView int   // the latest recieved view number
	nodes      int   // the nodes number in system
}

// NewClient: create a new client
// params:
// - addr: the address of client
// - port: the port of client
func NewClient(addr string, port string) *Client {
	sk, pk, err := sm2.Sm2KeyGen(rand.Reader)
	if err != nil {
		return nil
	}
	// go ssm2.WriteKey(pk, "../../config/client/public.pem")
	// go ssm2.WriteKey(sk, "../../config/client/private.pem")
	client := &Client{
		identity.PrivID{
			ID: identity.PubID{
				Name:   "client",
				PubKey: pk,
			},
			Address:    nil,
			PrivateKey: sk,
		},
		addr,
		port,
		"127.0.0.1:20000",
		map[int]string{0: "20001", 1: "20002", 2: "20003", 3: "20004"},
		make(map[int][]interface{}),
		*log.New(os.Stdout, "", 0),
		sync.Mutex{},
		0, 0, 1, 0, 0, -1, 0, 0, 0,
	}

	client.Logger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	return client
}

// StartClient: start the client to connect to the server and wait the connect form servers
func (c *Client) StartClient() {
	conn, err := net.Dial("tcp", c.SerAddr)
	if err != nil {
		c.Logger.Println("Error connecting to server:", err.Error())
		return
	}
	defer conn.Close()

	go c.StartListenTcp()
	reader := bufio.NewReader(os.Stdin)

	// count := 100000
	// path := "../../config/client/"

	// wst := time.Now().UnixNano() / 1e6
	// GenNewReqs(path, count, 1024)
	// wet := time.Now().UnixNano() / 1e6
	// fmt.Println(float64(wet-wst)/float64(count), "ms")

	// rst := time.Now().UnixNano() / 1e6
	// factory.ReadReq(count)
	// ret := time.Now().UnixNano() / 1e6
	// fmt.Println(float64(ret-rst)/float64(count), "ms")

	for {
		msg, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)

		// now := time.Now().UnixNano() / 1e6
		// send message to server
		if len(msg) != 0 {
			conn.Write([]byte(msg + "\n"))
			c.viewCount = 0
			c.Logger.Println(msg)
			if msg[0] == 'a' {
				input := strings.Split(msg, " ")
				if len(input) < 4 {
					c.reqNum = 10
				} else {
					for i := range input[1 : len(input)-1] {
						val, err := strconv.Atoi(input[i+1])
						if err == nil {
							c.reqNum *= val
						} else {
							c.reqNum *= i
						}
					}
					val, err := strconv.Atoi(input[len(input)-2])
					if err == nil {
						c.batchSize = val
					}
				}
			} else if msg[0] == 'q' {
				// fmt.Println(12312312)
				// q means quit the system
				break
			}
		} else {
			c.LogResult()
			c.StoreResult()
			c.RefreshState()
			// break
		}
		// c.startTime = now
	}
}

func (c *Client) StartListenTcp() {
	ln, err := net.Listen("tcp", ":"+c.Port)
	if err != nil {
		c.Logger.Println("Error listening:", err.Error())
		return
	}
	defer ln.Close()

	c.Logger.Println("Client started on port", c.Port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			c.Logger.Println("Error accepting connection:", err.Error())
			continue
		}
		go c.handleConn(conn)
	}
}

// handleConn: handle the server connect to the client
func (c *Client) handleConn(conn net.Conn) {
	// c.Logger.Println("Client connected:", conn.RemoteAddr())
	defer conn.Close()
	for {

		// message length < 2048
		buf := make([]byte, 2048)
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				c.Logger.Println("Error reading:", err)
			}
			return
		}

		msg := &message.ServerMsg{}
		err = json.Unmarshal(buf[:n], msg)

		if err != nil {
			c.Logger.Println("Error Json Unmarshal in handleConn", err.Error())
		}
		go c.handleReply(msg)
	}
}

// handleReply: handle the reply of request from the servers
func (c *Client) handleReply(msg *message.ServerMsg) {
	now := time.Now().UnixNano() / 1e6
	if msg.SendServer == "start" {
		c.startTime = now
		c.endTime = now
		c.nodes = int(msg.Payload[0])
		return
	}
	hsMsg := &hstypes.Msg{}
	err := json.Unmarshal(msg.Payload, hsMsg)
	if err != nil {
		c.Logger.Println("Error Json Unmarshal in HandleReply", err.Error())
	}
	c.endTime = now
	c.endView = hsMsg.ViewNumber
	if c.startView == -1 {
		c.startView = hsMsg.ViewNumber
	}

	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.showedView < hsMsg.ViewNumber {
		d, co, s := dcs.GetDCS(c.nodes, float64(c.endTime-c.startTime)/(1000*float64(c.endView-c.startView+1)), float64(c.batchSize*(c.endView-c.startView+1)*1000)/float64(c.endTime-c.startTime))
		fmt.Printf("Decentralization: %.4f, Consistency: %.4f, Scalability: %.4f \n", d, co, s)
		c.showedView = hsMsg.ViewNumber
	}
}

// LogResult: log the six result
func (c *Client) LogResult() {
	c.Logger.Printf("Reqest number: %d\n", c.batchSize*(c.endView-c.startView+1))
	c.Logger.Printf("ReqBatchSize : %d\n", c.batchSize)
	c.Logger.Printf("Total time   : %d%s\n", c.endTime-c.startTime, "ms")
	c.Logger.Printf("View number  : %d\n", c.endView-c.startView+1)
	c.Logger.Printf("Throughput   : %d%s\n", c.reqNum*1000/int(c.endTime-c.startTime), "tps")
	c.Logger.Printf("Latency      : %d%s\n", int(c.endTime-c.startTime)/(c.endView-c.startView+1), "ms")
}

// StoreResult: store the result to file
func (c *Client) StoreResult() {
	// file name and worksheet name
	filename := "data_chained_hotstuff.xlsx"
	sheetName := "Batchsize" + strconv.Itoa(c.batchSize)

	// check whether the file exists
	var file *excelize.File
	if _, err := os.Stat(filename); os.IsNotExist(err) {

		// if the file does not exist, create a new file
		file = excelize.NewFile()
		rowKey := []interface{}{"Reqest number", "BatchSize", "Total time", "View number", "Throughput", "Latency"}
		if err := file.SetSheetRow(sheetName, fmt.Sprintf("A%d", 1), &rowKey); err != nil {
			fmt.Println("Error setting row:", err)
			return
		}
	} else {
		// if the file exists, open the existing file
		file, err = excelize.OpenFile(filename)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
	}

	// get existing data
	rows, err := file.GetRows(sheetName)
	if err != nil {
		fmt.Println("Error getting row:", err)
		return
	}

	// data to append
	data := []interface{}{
		c.batchSize * (c.endView - c.startView + 1),
		c.batchSize,
		c.endTime - c.startTime,
		c.endView - c.startView + 1,
		c.reqNum * 1000 / int(c.endTime-c.startTime),
		int(c.endTime-c.startTime) / (c.endView - c.startView + 1),
	} // new row data

	if err := file.SetSheetRow(sheetName, fmt.Sprintf("A%d", len(rows)+1), &data); err != nil {
		fmt.Println("Error setting row:", err)
		return
	}

	// save file
	if err := file.SaveAs(filename); err != nil {
		fmt.Println("Error saving file:", err)
		return
	}

	fmt.Println("Excel file saved successfully.")
}

// RefreshState: refresh the client state
func (c *Client) RefreshState() {
	c.startTime = 0
	c.endTime = 0
	c.reqNum = 1
	c.batchSize = 0
	c.viewCount = 0
	c.startView = -1
}
