package service

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	// "strings"

	// "log"
	// "math/big"

	token "github.com/DuongQuyen1309/indexevent"
	"github.com/DuongQuyen1309/indexevent/internal/datastore"

	// "github.com/ethereum/go-ethereum"
	// "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func IndexEvent() error {
	client, err := ConnectBSCNode("https://bsc-mainnet.nodereal.io/v1/cebf31df832245339f9655fd1a592797")
	if err != nil {
		fmt.Println("Error connect BSC node", err)
		return err
	}
	maxCurrentBlockHead, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return err
	}
	maxCurrentBlock := maxCurrentBlockHead.Number.Uint64()

	// var sink = make(chan *token.WheelRequestCreated)
	// _, err = constractInstance.WatchRequestCreated(&bind.WatchOpts{
	// 	Context: context.Background(),
	// 	Start:   &maxCurrentBlock,
	// }, sink, nil, nil)
	// if err != nil {
	// 	fmt.Println("Error watch request created", err)
	// 	return err
	// }

	// go func() {
	// 	for {
	// 		for newEvent := range sink {
	// 			header, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(newEvent.Raw.BlockNumber)))
	// 			if err != nil {
	// 				fmt.Println("Error get header in temp watch request created", err)
	// 				return
	// 			}
	// 			timestamp := time.Unix(int64(header.Time), 0)
	// 			hash := common.HexToHash(newEvent.Raw.Topics[1].Hex())
	// 			requestOwner := common.BytesToAddress(hash.Bytes()[12:])
	// 			datastore.InsertTempResquestCreatedDB(newEvent, requestOwner.String(), timestamp)
	// 		}
	// 	}
	// }()

	// go func() {
	// 	for {
	// 		var sink = make(chan *token.WheelRequestCreated)
	// 		_, err := constractInstance.WatchRequestCreated(&bind.WatchOpts{
	// 			Context: context.Background(),
	// 			Start:   &maxCurrentBlock,
	// 		}, sink, nil, nil)
	// 		if err != nil {
	// 			fmt.Println("Error watch request created", err)
	// 			return
	// 		}
	// 		for newEvent := range sink {
	// 			header, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(newEvent.Raw.BlockNumber)))
	// 			if err != nil {
	// 				fmt.Println("Error get header in temp watch request created", err)
	// 				return
	// 			}
	// 			timestamp := time.Unix(int64(header.Time), 0)
	// 			hash := common.HexToHash(newEvent.Raw.Topics[1].Hex())
	// 			requestOwner := common.BytesToAddress(hash.Bytes()[12:])
	// 			datastore.InsertTempResquestCreatedDB(newEvent, requestOwner.String(), timestamp)
	// 		}
	// 	}
	// }()

	constractInstance, err := token.NewWheelFilterer(common.HexToAddress("0x0DF49Ee109bE77DA53d3050575e409D28D542ECC"), client)
	if err != nil {
		fmt.Println("Error create contract instance", err)
		return err
	}
	var startBlock uint64 = 20977112
	endBlock := startBlock + 100
	for {
		var wg sync.WaitGroup
		var mu sync.Mutex
		wg.Add(2)
		go func(startBlock uint64, endBlock uint64) {
			defer wg.Done()
			err = CrawlRequestCreatedInRange(client, constractInstance, startBlock, endBlock)
			if err != nil {
				fmt.Println("Error crawl event", err)
				return
			}
		}(startBlock, endBlock)

		time.Sleep(200 * time.Millisecond)
		go func(startBlock uint64, endBlock uint64) {
			defer wg.Done()
			err = CrawlResponseCreatedInRange(client, constractInstance, startBlock, endBlock)
			if err != nil {
				fmt.Println("Error Crawl ResponseCreated In Range", err)
				return
			}
		}(startBlock, endBlock)
		wg.Wait()
		mu.Lock()
		startBlock = endBlock + 1
		endBlock = startBlock + 100
		if endBlock > maxCurrentBlock {
			time.Sleep(10 * time.Second)
		}
		mu.Unlock()
		maxCurrentBlockHead, err = client.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return err
		}
		maxCurrentBlock = maxCurrentBlockHead.Number.Uint64()
	}
}

func ConnectBSCNode(rpcUrl string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func CrawlRequestCreatedInRange(client *ethclient.Client, constractInstance *token.WheelFilterer, startBlock uint64, endBlock uint64) error {
	iter, err := constractInstance.FilterRequestCreated(&bind.FilterOpts{
		Start: startBlock,
		End:   &endBlock,
	}, nil, nil)
	if err != nil {
		fmt.Println("Error filter event", err)
		return err
	}
	for iter.Next() {
		log := iter.Event
		hash := common.HexToHash(log.Raw.Topics[1].Hex())
		requestOwner := common.BytesToAddress(hash.Bytes()[12:])
		header, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(log.Raw.BlockNumber)))
		if err != nil {
			return err
		}
		timestamp := time.Unix(int64(header.Time), 0)
		datastore.InsertResquestCreatedDB(log, requestOwner.String(), timestamp)
	}
	return nil
}
func CrawlResponseCreatedInRange(client *ethclient.Client, constractInstance *token.WheelFilterer, startBlock uint64, endBlock uint64) error {
	iter, err := constractInstance.FilterResponseCreated(&bind.FilterOpts{
		Start: startBlock,
		End:   &endBlock,
	}, nil, nil)
	if err != nil {
		fmt.Println("Error filter event", err)
		return err
	}
	for iter.Next() {
		log := iter.Event
		hash := common.HexToHash(log.Raw.Topics[1].Hex())
		requestOwner := common.BytesToAddress(hash.Bytes()[12:])
		header, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(log.Raw.BlockNumber)))
		if err != nil {
			return err
		}
		timestamp := time.Unix(int64(header.Time), 0)
		prizeIds := ConvertBigIntToInt(log.PrizeIds)
		//phai thu lai xem truy cap tung phan tu trong thuoc tinh dc luu thanh mang ok ko
		datastore.InsertResponseCreatedDB(log, prizeIds, requestOwner.String(), timestamp)
	}
	return nil
}
func ConvertBigIntToInt(prizeIds []*big.Int) []int64 {
	var result = make([]int64, 0)
	for _, id := range prizeIds {
		result = append(result, int64(id.Int64()))
	}
	return result
}
