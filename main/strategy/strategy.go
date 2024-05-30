package strategy

import (
	"context"
	"log"
	"fmt"
	"math/big"
	"bb/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func GetGasPrice(client *ethclient.Client) (*big.Int, error) {
	return client.SuggestGasPrice(context.Background())
}

// Helper function to find the appropriate rate from swap events (updated with live data from main.go)
func findRate(swapEvents []types.SwapEvent, baseToken, quoteToken, dexName string) (*big.Float, error) {
	for i := len(swapEvents) - 1; i >= 0; i-- {
		event := swapEvents[i]
		if event.Asset1Name == baseToken && event.Asset2Name == quoteToken && event.DEXName == dexName {
			return event.AmountOut.Amount1, nil
		}
		if event.Asset1Name == quoteToken && event.Asset2Name == baseToken && event.DEXName == dexName {
			return event.AmountOut.Amount2, nil
		}
	}
	return big.NewFloat(0), fmt.Errorf("    %s %s/%s (NA / NA)", dexName, baseToken, quoteToken)
}

func TradeArbitrageStrategy(ctx context.Context, client *ethclient.Client, pairs []types.Pair, swapEvents []types.SwapEvent) {
	log.Printf("Checking for arbitrage opportunities...")

	// build matrix
	matrix := buildMatrix(pairs, swapEvents, client)
  _ = matrix

	// Detect arbitrage opportunities using the Bellman-Ford algorithm
	//detectArbitrageOpportunity(matrix)

  // TODO - use matrix to represent paths between pairs then calculate k-cycles to detect arbitrage opportunities
}

func buildMatrix(pairs []types.Pair, swapEvents []types.SwapEvent, client *ethclient.Client) string {
	for _, pair := range pairs {
		p := pair

		rateForward, err := findRate(swapEvents, p.Asset1(), p.Asset2(), p.DEX())
		if err != nil {
			log.Printf("%v", err)
			continue
		}
		rateBackward, err := findRate(swapEvents, p.Asset2(), p.Asset1(), p.DEX())
		if err != nil {
			log.Printf("%v", err)
			continue
		}

    gasCost, err := GetGasPrice(client)
    if err != nil {
      log.Printf("%v", err)
    }

    _ = gasCost

		log.Printf("    %s %s/%s (%f / %f) (%s)", p.DEX(), p.Asset1(), p.Asset2(), rateForward, rateBackward, p.Address())

	}
	return "WIP"
}


func Announce() {
	log.Printf("x-dex x-token swap")
	log.Printf("")
}
