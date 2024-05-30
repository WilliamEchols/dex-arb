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

  "bb/types"
  "bb/strategy"

  "bb/contracts/uniswapv2"
  "bb/contracts/sushiswap"
)

var (
  swapEventChan = make(chan types.SwapEvent)
  wg            sync.WaitGroup
)

func main() {
  log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

  startLog()

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
    log.Println("connected to node")
  }

  // Swap pairs
  // TODO - represent this in .env
  // TODO - verify these pairs and their decimal numbers (last two arguments)
  // TODO - find as many pairs as possible, especially pairs that will complete cycles for arbitrage
  pairs := []types.Pair{
    // uniswapv2
    uniswapv2pair.NewInstance("0x0d4a11d5EEaaC28EC3F61d100daF4d40471f1852", client, PRIVATE_KEY, CHAIN_ID, "ETH", "USDT", 18, 6),
    uniswapv2pair.NewInstance("0xa478c2975ab1ea89e8196811f51a7b7ade33eb11", client, PRIVATE_KEY, CHAIN_ID, "DAI", "ETH", 6, 18),
    uniswapv2pair.NewInstance("0x004375Dff511095CC5A197A54140a24eFEF3A416", client, PRIVATE_KEY, CHAIN_ID, "WBTC", "USDC", 8, 6),
    uniswapv2pair.NewInstance("0xb4e16d0168e52d35cacd2c6185b44281ec28c9dc", client, PRIVATE_KEY, CHAIN_ID, "USDC", "ETH", 6, 18),
    uniswapv2pair.NewInstance("0x517f9dd285e75b599234f7221227339478d0fcc8", client, PRIVATE_KEY, CHAIN_ID, "DAI", "MKR", 18, 18),
    uniswapv2pair.NewInstance("0xbb2b8038a1640196fbe3e38816f3e67cba72d940", client, PRIVATE_KEY, CHAIN_ID, "WBTC", "ETH", 8, 18),
    uniswapv2pair.NewInstance("0xd3d2e2692501a5c9ca623199d38826e513033a17", client, PRIVATE_KEY, CHAIN_ID, "UNI", "ETH", 18, 18),
    uniswapv2pair.NewInstance("0x21b8065d10f73ee2e260e5b47d3344d3ced7596e", client, PRIVATE_KEY, CHAIN_ID, "LINK", "ETH", 18, 18),

    // sushiswap
    sushiswappair.NewInstance("0x06da0fd433c1a5d7a4faa01111c044910a184553", client, PRIVATE_KEY, CHAIN_ID, "ETH", "USDT", 18, 6),
    sushiswappair.NewInstance("0x397ff1542f962076d0bfe58ea045ffa2d347aca0", client, PRIVATE_KEY, CHAIN_ID, "USDC", "ETH", 6, 18),
    sushiswappair.NewInstance("0xceff51756c56ceffca006cd410b03ffc46dd3a58", client, PRIVATE_KEY, CHAIN_ID, "WBTC", "ETH", 8, 18),
    sushiswappair.NewInstance("0x088ee5007c98a9677165d78dd2109ae4a3d04d0c", client, PRIVATE_KEY, CHAIN_ID, "YFI", "ETH", 18, 18),
    sushiswappair.NewInstance("0x055dB9AFF4311788264798356bbF3a733AE181c6", client, PRIVATE_KEY, CHAIN_ID, "SUSHI", "ETH", 18, 18),
  }  

  // TODO - ensure connected to correct addresses (if possible and reasonable)

  strategy.Announce()

  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  for _, pair := range pairs {
    wg.Add(1)
    go pair.Monitor(ctx, swapEventChan)
  }

  go monitorProcesses(ctx, client, swapEventChan, pairs)

  wg.Wait()
}

func monitorProcesses(ctx context.Context, client *ethclient.Client, swapEventChan <-chan types.SwapEvent, pairs []types.Pair) {
  var swapEvents []types.SwapEvent

  for {
    select {
      case <-ctx.Done():
        return
      case swapEvent := <-swapEventChan:
        log.Printf("")
        log.Printf("Detected swap event for %s/%s on %s (%s)", swapEvent.Asset1Name, swapEvent.Asset2Name, swapEvent.DEXName, swapEvent.Address)

        // Store the swapEvent in the slice for history tracking
        swapEvents = append(swapEvents, swapEvent)

        // TODO - make this is interupptable in case of new swap event (indicating underlying price change)
        go strategy.TradeArbitrageStrategy(ctx, client, pairs, swapEvents)
    }
  }
}

func startLog() {
  log.Printf("              ")
  log.Printf("  _     _     ")
  log.Printf(" | |   | |    ")
  log.Printf(" | |__ | |__  ")
  log.Printf(" | '_ \\| '_ \\ ")
  log.Printf(" | |_) | |_) |")
  log.Printf(" |_.__/|_.__/ ")
  log.Printf("              ")
  log.Printf("   big bucks  ")
  log.Printf("              ")
  log.Printf("              ")
}
