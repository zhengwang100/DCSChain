# DCS Chain

## overview

**DCS Chain** is a lightweight blockchain platform that leverages the **DCS impossibility theorem**. Through dynamic balance adjustment, it aims to  optimize **Decentralization**, **Consistency** and **Scalability parameters of the system**, thereby showcasing superior blockchain performance. This is an example of implementing [DCS Chain](https://arxiv.org/abs/2406.12376).

- Decentralization

- Consistency

- Scalability

**Table of Contents**

1. **Data Layer**: This layer is mainly responsible for the transmission, modification and storage of data.
   - **Network Engine**: responsible for communication between nodes and data transmission
   - **Verifiable Struct**: responsible for creating a data structure that is reached by multiple requests. By giving a proof of each request, the originator of each request can verify that his request has been executed and packaged into a block to be submitted to the blockchain.
   - **Block Maker**: responsible for packaging creates a new block and sends it to the sequencer responsible for consensus for consensus. Finally submitted to the blockchain.
2. **Consensus Layer**: This layer is mainly responsible for the consensus-related content of Transaction, including the management of nodes participating in consensus, BFT consensus protocol and related security tools.
   - **BFT Consensus**: The optional consensus protocols used in this system include PBFT, HotStuff, HotStuff-2.
   - **Security Tools**: Cryptographic or other tools used throughout the operation of the system to ensure security and reliability.
3. **Application Layer**: This layer is mainly the application services that can be provided by this system, with security provided by the consensus layer, and there are many other.
   - **Smart Contracts**: For most of the businesses including decentralized finance.
   - **Source Trace**: For the traceability of certain supply chains.
   - **Digital Certificate**: including electronic notarization, intellectual property rights and other businesses.

<img src="E:\Typora\file_pictures\image-20240623174414295.png" alt="image-20240623174414295" style="zoom:50%;" />

4. **Additional mechanisms**
   - **DCS Strategy Coordinator**: Based on the DCS theory to carry out the strategy of dynamic adjustment of the relevant parameters in the system, so as to realize the system to achieve the optimal state on the DCS triangle.
   - **Decentralized Deploy**: Decentralized deployment makes our system quite fault-tolerant. It can also improve the scalability and availability of the system.
   - **Node Manager**: Responsible for maintaining the node table, dynamic joining and exiting of nodes based on log information, and maintenance of log information.
   - **Data Consistence**: The data of all nodes in the system remains highly consistent, and all honest nodes have the same data. Data consistency ensures that nodes can transition from the same state to the same state.

## Environment

The current environment is simple and can be easily adapted to both Windows and Linux. The only need is as follows

- Go >= 1.21.5

## Run and Use DCS Chain

Next, we show how to quickly use DCS Chain and deploy replicas on your local machine.

### Run DCS Chain Without client.

Here, we show how to run DCS Chaine without a client. There is no client but you can still enter the client commands that we implemented. You just need to continue typing client commands after running them.
It is worth noting that this creates the first creation block by default.

``` shell
go run /cmd/run_without_client/rwoc.go -pr protocal -n node_number -p path
```

- -pr: the protocal type, in the current version you can use four protocols, as follows:

  - bh: [basic-hotstuff](./consensus/hotstuff/README.md)

  - ch: [chained-hotstuff](./consensus/hotstuff/README.md) 
  
  - h2: [hotstuff-2](./consensus/hotstuff2/README.md)
  
  - pbft: [PBFT](./consensus/pbft/README.md)
  
  Note: Not recommended because of the high performance requirements of the computer.

- -n: the node number

  Following the principles of the BFT consensus protocol, you would ideally choose a number of nodes of $n=3f+1(4, 7, 10...)$, with f identifying the integer that maximizes the number of Byzantine nodes. In addition, if you use $3f+2$($3f+3...$) is pointless, as it does not improve fault tolerance or efficiency.

  Note: The default number of nodes is 4.
  
- -pa: the path of block storage

  This parameter is the path where each node stores the block. In order to facilitate inspection, we store the block in txt text. The files named by block height are stored in a folder named after the node name under the path. Each file is a block. 

  Note: The default path is "BCData" in the project root path.

#### Client Commands

When you successfully start running the consensus protocol, the first consensus is performed by default, as a pair of genesis blocks, so you really start all your commands from view 1.

- ```shell
  r <request>: 
  ```

  generate a new request and send to the leader, and request represents a specific request operation

- ``` shell
  a <count, req_num, length>
  ```

  auto generate new requests, three parameters are respectively defined as count, req_num, length

  - count: the number of requests sent
  - req_num: the number of operation commands contained in each request
  - length: Length of each command

- ```shel
  c
  ```

  check the chained node information


- ```shell
  j
  ```

  a new server join the system

- ```shell
  e <node_id>
  ```

  a server exit from the system

- ```shell
  b
  ```

  get how many blocks and how many requests are included in the current system

- ``` shell
  q
  ```

  quit the program

### Run With A Client

If you want to start a cluster of servers in client-side mode, you need to add additional commands.

You need to start the node on the server side and listen to a port. This port is responsible for receiving command requests from clients. The command request is then distributed to a real blockchain server.

```shell
go run /cmd/run_without_client/rwc.go -r role -po port -pr protocal -n node_number -p path
```

- -r: role, start a server cluster or a client. Parameters are as follows:

  - server: start the blockchain server that participates in the consensus. The current server does not receive requests entered directly from the command line, only commands sent from the client. It also receives a response request from the server.

  - client:  The Client can send Commands to the server with the same commands as the previous Client Commands.

- -po: the port monitored by the server or client

  The server and client listen on one port each. For simplicity, the server only listens on one port.


There are three other parameters as shown in the previous section.
