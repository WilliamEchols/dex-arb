package strategy

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/big"
	"bb/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type AssetDEX struct {
	Asset string
	DEX   string
}

func GetGasPrice(client *ethclient.Client) (*big.Int, error) {
	return client.SuggestGasPrice(context.Background())
}

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
	return nil, fmt.Errorf("  - %s %s/%s (NA / NA)", dexName, baseToken, quoteToken)
}

func TradeArbitrageStrategy(ctx context.Context, client *ethclient.Client, pairs []types.Pair, swapEvents []types.SwapEvent) {
	log.Printf("Checking for arbitrage opportunities...")

	matrix := buildMatrix(pairs, swapEvents)
	detectArbitrageOpportunity(matrix, client)

	select {
	case <-ctx.Done():
		log.Println("TradeArbitrageStrategy interrupted before trade execution")
		return
	default:
	}
}

func buildMatrix(pairs []types.Pair, swapEvents []types.SwapEvent) map[AssetDEX]map[AssetDEX]*big.Float {
	matrix := make(map[AssetDEX]map[AssetDEX]*big.Float)

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

		log.Printf("  - %s %s/%s (%f / %f) (%s)", p.DEX(), p.Asset1(), p.Asset2(), rateForward, rateBackward, p.Address())

		fromAssetDEX := AssetDEX{p.Asset1(), p.DEX()}
		toAssetDEX := AssetDEX{p.Asset2(), p.DEX()}

		if matrix[fromAssetDEX] == nil {
			matrix[fromAssetDEX] = make(map[AssetDEX]*big.Float)
		}
		if matrix[toAssetDEX] == nil {
			matrix[toAssetDEX] = make(map[AssetDEX]*big.Float)
		}

		matrix[fromAssetDEX][toAssetDEX] = rateForward
		matrix[toAssetDEX][fromAssetDEX] = rateBackward
	}

	for assetDEX1 := range matrix {
		for assetDEX2 := range matrix {
			if assetDEX1.Asset == assetDEX2.Asset && assetDEX1.DEX != assetDEX2.DEX {
				if matrix[assetDEX1] == nil {
					matrix[assetDEX1] = make(map[AssetDEX]*big.Float)
				}
				if matrix[assetDEX2] == nil {
					matrix[assetDEX2] = make(map[AssetDEX]*big.Float)
				}
				matrix[assetDEX1][assetDEX2] = big.NewFloat(1)
				matrix[assetDEX2][assetDEX1] = big.NewFloat(1)
			}
		}
	}

	return matrix
}

func detectArbitrageOpportunity(matrix map[AssetDEX]map[AssetDEX]*big.Float, client *ethclient.Client) {
	graph := buildGraph(matrix)
	distances, predecessors := bellmanFord(graph, len(graph))

	for i := range graph {
		for j := range graph[i] {
			if distances[j] > distances[i]+graph[i][j] {
				log.Printf("Arbitrage opportunity detected!")
				logPath(predecessors, i, j)
				return
			}
		}
	}

	log.Println("No arbitrage opportunity detected.")
}

func buildGraph(matrix map[AssetDEX]map[AssetDEX]*big.Float) [][]float64 {
	n := len(matrix)
	graph := make([][]float64, n)
	for i := range graph {
		graph[i] = make([]float64, n)
		for j := range graph[i] {
			graph[i][j] = math.Inf(1)
		}
	}

	nodes := make(map[AssetDEX]int)
	i := 0
	for assetDEX1 := range matrix {
		nodes[assetDEX1] = i
		i++
	}

	for assetDEX1, edges := range matrix {
		for assetDEX2, rate := range edges {
			graph[nodes[assetDEX1]][nodes[assetDEX2]] = negativeLog(rate)
		}
	}

	return graph
}

func negativeLog(rate *big.Float) float64 {
	rateFloat, _ := rate.Float64()
	if rateFloat <= 0 {
		log.Printf("Invalid rate value for logarithm: %f", rateFloat)
		return math.Inf(1) // Treat invalid rates as infinite cost
	}
	return -math.Log(rateFloat)
}

func bellmanFord(graph [][]float64, n int) ([]float64, []int) {
	distances := make([]float64, n)
	predecessors := make([]int, n)
	for i := range distances {
		distances[i] = math.Inf(1)
		predecessors[i] = -1
	}
	distances[0] = 0

	for k := 0; k < n-1; k++ {
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if distances[j] > distances[i] + graph[i][j] {
					distances[j] = distances[i] + graph[i][j]
					predecessors[j] = i
				}
			}
		}
	}

	return distances, predecessors
}

func logPath(predecessors []int, start, end int) {
	path := []int{end}
	for predecessors[end] != -1 {
		path = append([]int{predecessors[end]}, path...)
		end = predecessors[end]
	}
	log.Printf("Arbitrage path: %v", path)
}

func Announce() {
	log.Printf("x-dex x-token arb")
}
