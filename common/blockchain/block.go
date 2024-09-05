package blockchain

import (
	common "common"
	"encoding/json"
	"fmt"
	"merkle"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// BlockStore: the storage core of the blockchain is responsible for the reading and writing of the blockchain
type BlockStore struct {
	Base            int64  // reserved field
	Height          int    // the height of form a new block or current block
	GeneratedHeight int    // the height of the generated block,mainly used for Chained-Hotstuff
	PreBlkHash      []byte // the hash of previous block
	CurBlkHash      []byte // the hash of current block
	CurProposalBlk  Block  // the block of current proposal in current view
	Path            string // the storage path of the block
	WMu             sync.Mutex
}

// Block: block entity, Block = BlockHeader + BlockData
type Block struct {
	BlkHdr  BlockHeader // the header of block
	BlkData BlockData   // the data of block
}

// BlockHeader: the header of a block, which include some information about block but no concrete transactions
type BlockHeader struct {
	Height      int    // the height of the block
	ViewNumber  int    // the view number when the block is presented
	TimeStamp   int64  // the timestamp when the block is presented
	PreBlkHash  []byte // thr hash of the previous block hash
	RootHash    []byte // the root hash of merkel tree consisting of all transactions in the block
	Validation  []byte // the signature of the block of 2f+1 nodes
	BlkDataHash []byte // the hash of the block data
}

// BlockData: the data body of a block, which include concrete transctions and necessary information
type BlockData struct {
	Height   int      // the height of the block
	RootHash []byte   // the root hash of merkel tree consisting of all transactions in the block
	Trans    []string // the block contains concrete transctions
}

// WriteBlock: wirte current block to local and update the height
// params:
// - path: the path of block storage, file name is the height of block
// - blk: the block to be stored
// return:
// - error
func (bs *BlockStore) WriteBlock(path string, blk Block) error {
	// fmt.Println(blk.BlkData.Trans[0])
	BlkHdrJson, err := json.Marshal(blk)

	if err != nil {
		return err
	}
	err = os.WriteFile(path+strconv.Itoa(blk.BlkHdr.Height)+".txt", BlkHdrJson, 0644)
	if err != nil {
		return err
	} else {
		bs.Height += 1
		return nil
	}
}

// ReadBlock: read a block of a specified height from path
// params:
// - path: the path of block read
// - blkHeight: the height of block
// - return block and error
func (bs *BlockStore) ReadBlock(path string, blkHeight int) (*Block, error) {
	content, err := os.ReadFile(path + strconv.Itoa(blkHeight) + ".txt")
	if err != nil {
		panic(err)
	}
	var Blk Block
	// var BlkData BlockData
	err = json.Unmarshal(content, &Blk)
	if err != nil {
		panic(err)
	}
	return &Blk, nil
}

// GetBlockNamesSlice:  by traversing the path stored in the block
// return the list of filename
func GetBlockNamesSlice(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(filePath string, info os.FileInfo, err error) error {
		_, filename := filepath.Split(filePath)
		fileprefix := filename[0 : len(filename)-len(path.Ext(filename))]
		files = append(files, fileprefix)
		return nil
	})
	return files[1:], err
}

// GetBlockHeight: calculate the block height
// return the BlockHeight(int64) in the local files and error
func GetBlockHeight(root string) (int64, error) {

	// get the local block names which is the height of block
	height := 0
	err := filepath.Walk(root, func(filePath string, info os.FileInfo, err error) error {
		_, filename := filepath.Split(filePath)
		fileprefix := filename[0 : len(filename)-len(path.Ext(filename))]
		if h, err := strconv.Atoi(fileprefix); h > height && err == nil {
			height = h
		}
		return nil
	})
	if err != nil {
		return -1, err
	}

	// return the name or height of the last block
	BlockHeight, err := strconv.ParseInt(strconv.Itoa(height), 10, 64)
	return BlockHeight, err
}

// Hash: get the block data hash
func (bd *BlockData) Hash() []byte {
	bdHash := make([]byte, 0)
	bdHash = append(bdHash, byte(bd.Height))
	bdHash = append(bdHash, bd.RootHash...)
	bdHash = append(bdHash, common.StringSlice2OneDimByteSlice(bd.Trans)...)
	return merkle.Sum(bdHash)
}

// Hash: get the block hash
// !!! the hash the entire block
func (b *Block) Hash() []byte {
	bHash := make([]byte, 0)
	bHash = append(bHash, byte(b.BlkHdr.Height))
	bHash = append(bHash, byte(b.BlkHdr.ViewNumber))
	bHash = append(bHash, byte(b.BlkHdr.TimeStamp))
	bHash = append(bHash, b.BlkHdr.PreBlkHash...)
	bHash = append(bHash, b.BlkHdr.RootHash...)
	// bHash = append(bHash, b.BlkHdr.Validation...)
	bHash = append(bHash, b.BlkHdr.BlkDataHash...)
	bHash = append(bHash, byte(b.BlkData.Height))
	bHash = append(bHash, b.BlkHdr.RootHash...)
	bHash = append(bHash, common.StringSlice2OneDimByteSlice(b.BlkData.Trans)...)
	return merkle.Sum(bHash)
}

// WirteBlock: write current the lastest node's block to local blockchain and refresh current block state
// params:
// - blk: the block to be stored
func (bs *BlockStore) StoreBlock(blk Block) {
	bs.WMu.Lock()
	defer bs.WMu.Unlock()
	for {
		dirPath := bs.Path
		_, err := os.Stat(dirPath)

		if os.IsNotExist(err) {
			os.MkdirAll(dirPath, 0755)
		} else if err != nil {
			fmt.Println("error occur:", err)
		} else {
			err := bs.WriteBlock(dirPath+"/", blk)
			if err == nil {
				break
			} else {
				fmt.Println("write block error", err)
			}
		}
	}
	bs.PreBlkHash = bs.CurBlkHash
	bs.CurBlkHash = nil
}

// GenNewBlock: generate a new block and assign it to bs
// params:
// - viewNumber: the view number when the block is generated
// - commands: the block contains commands/transctions
func (bs *BlockStore) GenNewBlock(viewNumber int, commands []string, heights ...int) {
	bs.WMu.Lock()
	defer bs.WMu.Unlock()

	newHeight := bs.Height

	if len(heights) > 0 {
		newHeight = heights[0]
		bs.GeneratedHeight += 1
	}

	newBlock := Block{
		BlkData: BlockData{
			Height:   newHeight,
			RootHash: merkle.HashFromByteSlices(common.StringSlice2TwoDimByteSlice(commands)),
			Trans:    commands,
		},
		BlkHdr: BlockHeader{
			Height:     newHeight,
			PreBlkHash: bs.PreBlkHash,
			ViewNumber: viewNumber,
			TimeStamp:  time.Now().UnixNano() / int64(time.Millisecond),
		},
	}
	newBlock.BlkHdr.RootHash = newBlock.BlkData.RootHash
	bs.CurProposalBlk = newBlock
	bs.CurProposalBlk.BlkHdr.BlkDataHash = bs.CurProposalBlk.BlkData.Hash()
	bs.CurBlkHash = bs.CurProposalBlk.Hash()
}

// GenEmptyBlock: generate an empty block
func (bs *BlockStore) GenEmptyBlock() {
	bs.WMu.Lock()
	defer bs.WMu.Unlock()

	bs.CurProposalBlk = Block{BlkHdr: BlockHeader{}, BlkData: BlockData{}}
	bs.CurBlkHash = merkle.EmptyHash()
}

// IsEmpty: determine whether the block is empty by the number of commands contained in the block
func (b *Block) IsEmpty() bool {
	return len(b.BlkData.Trans) == 0
}
