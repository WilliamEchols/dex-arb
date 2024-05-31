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

  strategy.Announce()

  // WebSocket connection
  client, err := ethclient.Dial(WSS_URL)
  if err != nil {
    log.Fatal(err)
  } else {
    log.Println("connected to node")
    log.Println("")
  }

  // Swap pairs
  // TODO - represent this in .env
  // TODO - verify these pairs and their decimal numbers (last two arguments)
  // TODO - find as many pairs as possible, especially pairs that will complete cycles for arbitrage
  pairs := []types.Pair{
    // uniswapv2
    uniswapv2pair.NewInstance("0x0d4a11d5EEaaC28EC3F61d100daF4d40471f1852", client, PRIVATE_KEY, CHAIN_ID, "ETH", "USDT", 18, 6),
    uniswapv2pair.NewInstance("0xa478c2975ab1ea89e8196811f51a7b7ade33eb11", client, PRIVATE_KEY, CHAIN_ID, "DAI", "ETH", 18, 18),
    uniswapv2pair.NewInstance("0x004375Dff511095CC5A197A54140a24eFEF3A416", client, PRIVATE_KEY, CHAIN_ID, "WBTC", "USDC", 8, 6),
    uniswapv2pair.NewInstance("0xb4e16d0168e52d35cacd2c6185b44281ec28c9dc", client, PRIVATE_KEY, CHAIN_ID, "USDC", "ETH", 6, 18),
    uniswapv2pair.NewInstance("0x517f9dd285e75b599234f7221227339478d0fcc8", client, PRIVATE_KEY, CHAIN_ID, "DAI", "MKR", 18, 18),
    uniswapv2pair.NewInstance("0xbb2b8038a1640196fbe3e38816f3e67cba72d940", client, PRIVATE_KEY, CHAIN_ID, "WBTC", "ETH", 8, 18),
    uniswapv2pair.NewInstance("0xd3d2e2692501a5c9ca623199d38826e513033a17", client, PRIVATE_KEY, CHAIN_ID, "UNI", "ETH", 18, 18),
    uniswapv2pair.NewInstance("0x21b8065d10f73ee2e260e5b47d3344d3ced7596e", client, PRIVATE_KEY, CHAIN_ID, "LINK", "ETH", 18, 18),
    uniswapv2pair.NewInstance("0x4B1F1e2435A9C96f7330FAea190Ef6A7C8D70001", client, PRIVATE_KEY, CHAIN_ID, "DAI", "USDT", 18, 6),
    uniswapv2pair.NewInstance("0x3041cbd36888becc7bbcbc0045e3b1f144466f5f", client, PRIVATE_KEY, CHAIN_ID, "USDC", "USDT", 6, 6),
    uniswapv2pair.NewInstance("0xdf0a1bb2A0a63b79F8ba774d25b887f1653c4ff5", client, PRIVATE_KEY, CHAIN_ID, "AAVE", "ETH", 18, 18),
    uniswapv2pair.NewInstance("0xcffdded873554f362ac02f8fb1f02e5ada10516f", client, PRIVATE_KEY, CHAIN_ID, "COMP", "ETH", 18, 18),
    uniswapv2pair.NewInstance("0xc2adda861f89bbb333c90c492cb837741916a225", client, PRIVATE_KEY, CHAIN_ID, "MANA", "ETH", 18, 18),
    uniswapv2pair.NewInstance("0x43ae24960e5534731fc831386c07755a2dc33d47", client, PRIVATE_KEY, CHAIN_ID, "SNX", "ETH", 18, 18),
    uniswapv2pair.NewInstance("0xb6909b960dbbe7392d405429eb2b3649752b4838", client, PRIVATE_KEY, CHAIN_ID, "BAT", "ETH", 18, 18),

    // sushiswap
    sushiswappair.NewInstance("0x06da0fd433c1a5d7a4faa01111c044910a184553", client, PRIVATE_KEY, CHAIN_ID, "ETH", "USDT", 18, 6),
    sushiswappair.NewInstance("0x397ff1542f962076d0bfe58ea045ffa2d347aca0", client, PRIVATE_KEY, CHAIN_ID, "USDC", "ETH", 6, 18),
    sushiswappair.NewInstance("0xceff51756c56ceffca006cd410b03ffc46dd3a58", client, PRIVATE_KEY, CHAIN_ID, "WBTC", "ETH", 8, 18),
    sushiswappair.NewInstance("0x088ee5007c98a9677165d78dd2109ae4a3d04d0c", client, PRIVATE_KEY, CHAIN_ID, "YFI", "ETH", 18, 18),
    sushiswappair.NewInstance("0x055dB9AFF4311788264798356bbF3a733AE181c6", client, PRIVATE_KEY, CHAIN_ID, "SUSHI", "ETH", 18, 18),
    sushiswappair.NewInstance("0x904f60E731DfD2fcfA674d5CC5E3d1D47E21c59b", client, PRIVATE_KEY, CHAIN_ID, "DAI", "USDT", 18, 6),
    sushiswappair.NewInstance("0x985458e523db3d53125813ed68c274899e9dfab4", client, PRIVATE_KEY, CHAIN_ID, "USD", "USDT", 6, 6),
    sushiswappair.NewInstance("0xA478c2975Ab1ea89e8196811F51A7b7Ade33eB11", client, PRIVATE_KEY, CHAIN_ID, "AAVE", "ETH", 18, 18),
    sushiswappair.NewInstance("0x31503dcb60119a812fee820bb7042752019f2355", client, PRIVATE_KEY, CHAIN_ID, "COMP", "ETH", 18, 18),
    sushiswappair.NewInstance("0x1c1D6E4F4a2E86A6C7686A04E6D48cA452B161B9", client, PRIVATE_KEY, CHAIN_ID, "MANA", "ETH", 18, 18),
    sushiswappair.NewInstance("0x43AE24960e5534731Fc831386c07755A2DC33D47", client, PRIVATE_KEY, CHAIN_ID, "SNX", "ETH", 18, 18),
    sushiswappair.NewInstance("0x0D8775F648430679A709E98d2b0Cb6250d2887EF", client, PRIVATE_KEY, CHAIN_ID, "BAT", "ETH", 18, 18),
  }

  // TODO - ensure connected to correct addresses (if possible and reasonable)

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
	var wg sync.WaitGroup
	tradeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			wg.Wait() // Wait for all goroutines to finish before returning
			return
		case swapEvent := <-swapEventChan:
      log.Printf("")
			log.Printf("Detected swap event for %s/%s on %s (%s)", swapEvent.Asset1Name, swapEvent.Asset2Name, swapEvent.DEXName, swapEvent.Address)

			// Store the swapEvent in the slice for history tracking
			swapEvents = append(swapEvents, swapEvent)

			// Cancel any ongoing trade execution and start a new one
			cancel()
			wg.Wait() // Wait for the previous goroutine to finish
			tradeCtx, cancel = context.WithCancel(ctx)

			wg.Add(1)
			go func() {
				defer wg.Done()
				strategy.TradeArbitrageStrategy(tradeCtx, client, pairs, swapEvents)
			}()
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
