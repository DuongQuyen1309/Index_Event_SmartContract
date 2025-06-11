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

var (
	wg sync.WaitGroup
)

func IndexEvent(ctx context.Context) error {
	//client để crawl các event trong quá khứ
	httpClient, err := ConnectBSCNode("https://bsc-mainnet.nodereal.io/v1/cebf31df832245339f9655fd1a592797")
	if err != nil {
		fmt.Println("Error connect BSC node", err)
		return err
	}
	maxCurrentBlockHead, err := httpClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	maxCurrentBlock := maxCurrentBlockHead.Number.Uint64()
	//constractInstance để crawl các event trong quá khứ
	constractInstance, err := token.NewWheelFilterer(common.HexToAddress("0x0DF49Ee109bE77DA53d3050575e409D28D542ECC"), httpClient)
	if err != nil {
		fmt.Println("Error create contract instance", err)
		return err
	}
	//client để sử dụng websocket để watch các event realtime
	wssClient, err := ConnectBSCNode("wss://bsc-mainnet.nodereal.io/ws/v1/cebf31df832245339f9655fd1a592797")
	if err != nil {
		fmt.Println("Error connect BSC node websocket", err)
		return err
	}
	//constractInstance để gọi watch event realtime
	realtimeConstractInstance, err := token.NewWheelFilterer(common.HexToAddress("0x0DF49Ee109bE77DA53d3050575e409D28D542ECC"), wssClient)
	if err != nil {
		fmt.Println("Error create contract instance for realtime", err)
		return err
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		pastTime, cancel := context.WithCancel(ctx)
		err = CrawlInPast(pastTime, cancel, constractInstance, httpClient, maxCurrentBlock)
		if err != nil {
			fmt.Println("Error crawl in past", err)
			return
		}
	}()
	go func() {
		defer wg.Done()
		realTime, cancel := context.WithCancel(ctx)
		WatchEventInRealtime(realTime, cancel, realtimeConstractInstance, httpClient, wssClient, maxCurrentBlock)
	}()
	wg.Wait()
	return nil
}
func WatchEventInRealtime(realTime context.Context, cancel context.CancelFunc, realtimeConstractInstance *token.WheelFilterer, client *ethclient.Client, wssClient *ethclient.Client, maxCurrentBlock uint64) {
	errChan := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := WatchRequestCreatedInRealtime(realTime, realtimeConstractInstance, client, maxCurrentBlock)
		if err != nil {
			errChan <- err
			fmt.Println("Error watch request created in realtime", err)
			return
		}
	}()
	go func() {
		defer wg.Done()
		err := WatchResponseCreatedInRealtime(realTime, realtimeConstractInstance, wssClient, maxCurrentBlock)
		if err != nil {
			errChan <- err
			fmt.Println("Error watch request created in realtime", err)
			return
		}
	}()
	select {
	case <-realTime.Done():
		cancel()
	case err := <-errChan:
		fmt.Println("Error in watching event", err)
		cancel()
	}
	wg.Wait()
}
func WatchResponseCreatedInRealtime(realTime context.Context, realtimeConstractInstance *token.WheelFilterer, client *ethclient.Client, maxCurrentBlock uint64) error {
	var sink = make(chan *token.WheelResponseCreated)
	_, err := realtimeConstractInstance.WatchResponseCreated(&bind.WatchOpts{
		Context: realTime,
		Start:   &maxCurrentBlock,
	}, sink, nil, nil)
	if err != nil {
		fmt.Println("Error watch request created", err)
		return err
	}
	for {
		select {
		case <-realTime.Done():
			return realTime.Err()
		case event, ok := <-sink:
			if !ok {
				err := fmt.Errorf("event channel closed in realtime watch")
				fmt.Println(err)
				return err
			}
			header, err := client.HeaderByNumber(realTime, big.NewInt(int64(event.Raw.BlockNumber)))
			if err != nil {
				fmt.Println("Error get header by number", err)
				return err
			}
			timestamp := time.Unix(int64(header.Time), 0)
			hash := common.HexToHash(event.Raw.Topics[1].Hex())
			requestOwner := common.BytesToAddress(hash.Bytes()[12:])
			prizeIds := ConvertBigIntToInt(event.PrizeIds)
			datastore.InsertResponseCreatedDB(event, prizeIds, requestOwner.String(), timestamp)
		}
	}
}

func WatchRequestCreatedInRealtime(realTime context.Context, realtimeConstractInstance *token.WheelFilterer, client *ethclient.Client, maxCurrentBlock uint64) error {
	var sink = make(chan *token.WheelRequestCreated)
	_, err := realtimeConstractInstance.WatchRequestCreated(&bind.WatchOpts{
		Context: realTime,
		Start:   &maxCurrentBlock,
	}, sink, nil, nil)
	if err != nil {
		fmt.Println("Error watch request created", err)
		return err
	}
	for {
		select {
		case <-realTime.Done():
			return realTime.Err()
		case event, ok := <-sink:
			if !ok {
				err := fmt.Errorf("event channel closed in realtime watch")
				fmt.Println(err)
				return err
			}
			header, err := client.HeaderByNumber(realTime, big.NewInt(int64(event.Raw.BlockNumber)))
			if err != nil {
				fmt.Println("Error get header by number", err)
				return err
			}
			timestamp := time.Unix(int64(header.Time), 0)
			hash := common.HexToHash(event.Raw.Topics[1].Hex())
			requestOwner := common.BytesToAddress(hash.Bytes()[12:])
			datastore.InsertResquestCreatedDB(event, requestOwner.String(), timestamp)
		}
	}
}

func CrawlInPast(pastTime context.Context, cancel context.CancelFunc, constractInstance *token.WheelFilterer, client *ethclient.Client, maxCurrentBlock uint64) error {
	errChan := make(chan error, 2)
	var startBlock uint64 = 20977112
	endBlock := startBlock + 100
	for {
		// var mu sync.Mutex
		wg.Add(2)
		go func(startBlock uint64, endBlock uint64) {
			defer wg.Done()
			err := CrawlRequestCreatedInRange(pastTime, client, constractInstance, startBlock, endBlock)
			if err != nil {
				errChan <- err
				fmt.Println("Error crawl request created", err)
				return
			}
		}(startBlock, endBlock)
		time.Sleep(200 * time.Millisecond)
		go func(startBlock uint64, endBlock uint64) {
			defer wg.Done()
			err := CrawlResponseCreatedInRange(pastTime, client, constractInstance, startBlock, endBlock)
			if err != nil {
				errChan <- err
				fmt.Println("Error crawl response created", err)
				return
			}
		}(startBlock, endBlock)
		select {
		case <-pastTime.Done():
			cancel()
		case err := <-errChan:
			fmt.Println("Error in watching event", err)
			cancel()
		}
		wg.Wait()
		// mu.Lock()
		startBlock = endBlock + 1
		endBlock = startBlock + 100
		if endBlock > maxCurrentBlock {
			endBlock = maxCurrentBlock
		}
		// mu.Unlock()
	}
}

func ConnectBSCNode(rpcUrl string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func CrawlRequestCreatedInRange(pastTime context.Context, client *ethclient.Client, constractInstance *token.WheelFilterer, startBlock uint64, endBlock uint64) error {
	iter, err := constractInstance.FilterRequestCreated(&bind.FilterOpts{
		Start:   startBlock,
		End:     &endBlock,
		Context: pastTime,
	}, nil, nil)
	if err != nil {
		fmt.Println("Error filter event", err)
		return err
	}
	select {
	case <-pastTime.Done():
		return pastTime.Err()
	default:
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
	}
	return nil
}
func CrawlResponseCreatedInRange(pastTime context.Context, client *ethclient.Client, constractInstance *token.WheelFilterer, startBlock uint64, endBlock uint64) error {
	iter, err := constractInstance.FilterResponseCreated(&bind.FilterOpts{
		Start:   startBlock,
		End:     &endBlock,
		Context: pastTime,
	}, nil, nil)
	if err != nil {
		fmt.Println("Error filter event", err)
		return err
	}
	select {
	case <-pastTime.Done():
		return pastTime.Err()
	default:
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
			datastore.InsertResponseCreatedDB(log, prizeIds, requestOwner.String(), timestamp)
		}
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
