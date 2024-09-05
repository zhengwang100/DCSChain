package blockchain_test

import (
	bc "blockchain"
	"fmt"
	"strconv"
	"testing"
)

// TestFileReadAndWrite: try to wirte blocks and read
func TestFileReadAndWirte(t *testing.T) {
	testBS := bc.BlockStore{
		Base:   64,
		Height: 0,
	}
	for i := 0; i < 10; i++ {
		blkData := bc.BlockData{
			Height:   testBS.Height,
			RootHash: []byte{byte(i), byte(i + 1), byte(i + 2)},
			Trans:    []string{"qwer", "asdf", "zxcv"},
		}
		blkHdr := bc.BlockHeader{
			Height:      testBS.Height,
			PreBlkHash:  []byte{byte(i), byte(i + 1), byte(i + 2)},
			RootHash:    []byte{byte(i), byte(i + 10), byte(i + 20)},
			Validation:  []byte{byte(i * 11), byte(i*11 + 1), byte(i*11 + 2)},
			BlkDataHash: []byte{byte(i * 22), byte(i*22 + 1), byte(i*22 + 2)},
		}
		testBS.CurProposalBlk = bc.Block{
			BlkHdr:  blkHdr,
			BlkData: blkData,
		}
		err := testBS.WriteBlock("./BCData/", testBS.CurProposalBlk)
		if err == nil {
			fmt.Println("write success")
		} else {
			panic(err)
		}
		blkR, err := testBS.ReadBlock("./BCData/", testBS.Height)
		if err != nil {
			panic(err)
		} else {
			fmt.Println(blkR)
			fmt.Println(blkR.BlkData)
			fmt.Println(blkR.BlkHdr)
		}
	}

}

// TestGetBlk: test the func GetBlockHeight
func TestGetBlk(t *testing.T) {
	blkNum, _ := bc.GetBlockHeight("../../BCData/r_0")
	fmt.Println(blkNum)
}

// TestStoreBlock: test store a block
func TestStoreBlock(t *testing.T) {
	testBS := bc.BlockStore{
		Base:   64,
		Height: 0,
		Path:   "./",
	}
	i := 1
	blkData := bc.BlockData{
		Height:   testBS.Height,
		RootHash: []byte{byte(i), byte(i + 1), byte(i + 2)},
		Trans:    []string{"qwer", "asdf", "zxcv"},
	}
	blkHdr := bc.BlockHeader{
		Height:      testBS.Height,
		PreBlkHash:  []byte{byte(i), byte(i + 1), byte(i + 2)},
		RootHash:    []byte{byte(i), byte(i + 10), byte(i + 20)},
		Validation:  []byte{byte(i * 11), byte(i*11 + 1), byte(i*11 + 2)},
		BlkDataHash: []byte{byte(i * 22), byte(i*22 + 1), byte(i*22 + 2)},
	}
	testBS.CurProposalBlk = bc.Block{
		BlkHdr:  blkHdr,
		BlkData: blkData,
	}
	testBS.StoreBlock(testBS.CurProposalBlk)
}

// TestStoreBlock: test generate a block
func TestGenerateBlock(t *testing.T) {
	testBS := bc.BlockStore{
		Base:            64,
		Height:          0,
		GeneratedHeight: 0,
		Path:            "./",
	}
	commands := []string{"1231111", "4561111", "789111"}
	for i := 0; i < 10; i++ {
		commands = append(commands, strconv.Itoa(i))
		testBS.GenNewBlock(i, commands)
		testBS.StoreBlock(testBS.CurProposalBlk)
		fmt.Println(i, testBS.CurProposalBlk.BlkHdr.Height)
	}
	fmt.Println("mid height", testBS.Height)
	for i := 0; i < 10; i++ {
		commands = append(commands, strconv.Itoa(i))
		testBS.GenNewBlock(i, commands, testBS.GeneratedHeight)
		// testBS.StoreBlock(testBS.CurProposalBlk)
		fmt.Println(i, testBS.CurProposalBlk.BlkHdr.Height)
	}

}
