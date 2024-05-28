package main

import (
  "context"
  "log"
  "os"
  "strconv"
  "sync"
  "math/big"

  "github.com/ethereum/go-ethereum/ethclient"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/joho/godotenv"

  "bb-e/types"
  "bb-e/strategy"

  "bb-e/contracts/uniswapv2"
)

var (
  swapEventChan = make(chan types.SwapEvent)
  wg            sync.WaitGroup
)

func main() {
  log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

  // Load environment variables
  err := godotenv.Load(".env.mainnet-test")
  if err != nil {
    log.Fatalf("Error loading .env file")
  }

  WSS_URL := os.Getenv("NODE_URL")

  PRIVATE_KEY, err := crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
  if err != nil {
    log.Fatalf("failed to parse private key: %v", err)
  }
  
  CHAIN_ID_INT64, err := strconv.ParseInt(os.Getenv("CHAIN_ID"), 10, 64)
  if err != nil {
    log.Fatalf("failed to parse chain id: %v", err)
  }
  CHAIN_ID := big.NewInt(CHAIN_ID_INT64)

  // WebSocket connection
  client, err := ethclient.Dial(WSS_URL)
  if err != nil {
    log.Fatal(err)
  } else {
    log.Println("Connected to Ethereum node")
  }

  // Swap pairs
  // TODO - represent this in .env
  // TODO - find more pairs
  pairs := []types.Pair{
    uniswapv2pair.NewInstance("0x0d4a11d5EEaaC28EC3F61d100daF4d40471f1852", client, PRIVATE_KEY, CHAIN_ID, "ETH", "USDT"),
    uniswapv2pair.NewInstance("0xa478c2975ab1ea89e8196811f51a7b7ade33eb11", client, PRIVATE_KEY, CHAIN_ID, "DAI", "ETH"),
    uniswapv2pair.NewInstance("0x004375Dff511095CC5A197A54140a24eFEF3A416", client, PRIVATE_KEY, CHAIN_ID, "WBTC", "USDC"),
    uniswapv2pair.NewInstance("0xb4e16d0168e52d35cacd2c6185b44281ec28c9dc", client, PRIVATE_KEY, CHAIN_ID, "USDC", "ETH"),
    uniswapv2pair.NewInstance("0x517f9dd285e75b599234f7221227339478d0fcc8", client, PRIVATE_KEY, CHAIN_ID, "DAI", "MKR"),
    uniswapv2pair.NewInstance("0xbb2b8038a1640196fbe3e38816f3e67cba72d940", client, PRIVATE_KEY, CHAIN_ID, "WBTC", "ETH"),

    // sushiswappair.NewInstance("0x06da0fd433c1a5d7a4faa01111c044910a184553", client, PRIVATE_KEY, CHAIN_ID, "ETH", "USDT")
    uniswapv2pair.NewInstance("0x06da0fd433c1a5d7a4faa01111c044910a184553", client, PRIVATE_KEY, CHAIN_ID, "ETH", "USDT"),
  }

  // TODO - ensure connected to correct addresses

  strategy.Announce()

  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  for _, pair := range pairs {
    wg.Add(1)
    go pair.Monitor(ctx, swapEventChan)
  }

  go monitorProcesses(ctx, swapEventChan, pairs)

  wg.Wait()
}

func monitorProcesses(ctx context.Context, swapEventChan <-chan types.SwapEvent, pairs []types.Pair) {
  var swapEvents []types.SwapEvent

  for {
    select {
      case <-ctx.Done():
        return
      case swapEvent := <-swapEventChan:
        log.Printf("Detected swap event for %s/%s on %s (%s)", swapEvent.Asset1Name, swapEvent.Asset2Name, swapEvent.DEXName, swapEvent.Address)

        // Store the swapEvent in the slice for history tracking
        swapEvents = append(swapEvents, swapEvent)

        // TODO - Matrix with swapEvent-entries representing exchange possibilities

        // TODO - make this is interupptable in case of new swap event (indicating underlying price change)
        go strategy.TradeArbitrageStrategy(ctx, pairs, swapEvents)
    }
  }
}

