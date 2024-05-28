package strategy

import (
	"context"
	"log"
  "fmt"
	"math"
	"math/big"

	"bb-e/types"
	"bb-e/contracts/uniswapv2"

	"github.com/ethereum/go-ethereum/ethclient"
)

func GetGasPrice(client *ethclient.Client) (*big.Int, error) {
	return client.SuggestGasPrice(context.Background())
}

func bigFloatToFloat64(f *big.Float) float64 {
	floatValue, _ := f.Float64()
	return floatValue
}

type TradingPair struct {
	BaseToken  string
	QuoteToken string
	DEXName    string
}

type Edge struct {
	From TradingPair
	To   TradingPair
	Rate float64
}

type Graph struct {
	Nodes map[TradingPair]struct{}
	Edges map[TradingPair][]Edge
}

func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[TradingPair]struct{}),
		Edges: make(map[TradingPair][]Edge),
	}
}

func (g *Graph) AddNode(pair TradingPair) {
	g.Nodes[pair] = struct{}{}
}

func (g *Graph) AddEdge(from, to TradingPair, rate float64) {
	edge := Edge{From: from, To: to, Rate: rate}
	g.Edges[from] = append(g.Edges[from], edge)
}

// Helper function to find the appropriate rate from swap events
func findRate(swapEvents []types.SwapEvent, baseToken, quoteToken, dexName string) (float64, error) {
	for _, event := range swapEvents {
		if event.Asset1Name == baseToken && event.Asset2Name == quoteToken && event.DEXName == dexName {
			return bigFloatToFloat64(new(big.Float).Quo(new(big.Float).SetInt(event.AmountOut.Amount1), big.NewFloat(1e18))), nil
		}
		if event.Asset1Name == quoteToken && event.Asset2Name == baseToken && event.DEXName == dexName {
			return bigFloatToFloat64(new(big.Float).Quo(new(big.Float).SetInt(event.AmountOut.Amount2), big.NewFloat(1e18))), nil
		}
	}
	return 0, fmt.Errorf("rate not found for %s/%s on %s", baseToken, quoteToken, dexName)
}

func TradeArbitrageStrategy(ctx context.Context, pairs []types.Pair, swapEvents []types.SwapEvent) {
	log.Printf("Checking for arbitrage opportunities...")

	graph := buildGraph(pairs, swapEvents)

	detectArbitrageOpportunities(graph)
}

func buildGraph(pairs []types.Pair, swapEvents []types.SwapEvent) *Graph {
	graph := NewGraph()

	for _, pair := range pairs {
		p := pair.(*uniswapv2pair.Instance)
    // p := pair.PairInstance // TODO - use something like this to not be uniswapv2pair specific
		tradingPair := TradingPair{BaseToken: p.Asset1Name, QuoteToken: p.Asset2Name, DEXName: p.DEXName}
		graph.AddNode(tradingPair)

    // TODO (Top priority) - fix this, both of these error out
    // TODO - I think an error may just indicate that we don't have a value for them in a swapEvent yet, this is expected
		rateForward, err := findRate(swapEvents, p.Asset1Name, p.Asset2Name, p.DEXName)
		if err != nil {
			log.Printf("Error finding rate: %v", err)
			continue
		}
		rateBackward, err := findRate(swapEvents, p.Asset2Name, p.Asset1Name, p.DEXName)
		if err != nil {
			log.Printf("Error finding rate: %v", err)
			continue
		}

    log.Printf("    forward rate: %f      backward rate: %f     addess: %s", rateForward, rateBackward, p.Address)

		graph.AddEdge(tradingPair, TradingPair{BaseToken: p.Asset2Name, QuoteToken: p.Asset1Name, DEXName: p.DEXName}, rateForward)
		graph.AddEdge(tradingPair, TradingPair{BaseToken: p.Asset1Name, QuoteToken: p.Asset2Name, DEXName: p.DEXName}, rateBackward)
	}

	// Add inter-exchange edges with rate 1.0 for equivalent tokens
	for _, pairA := range pairs {
		for _, pairB := range pairs {
			pA := pairA.(*uniswapv2pair.Instance)
			pB := pairB.(*uniswapv2pair.Instance)
			if pA.Asset1Name == pB.Asset1Name && pA.Asset2Name == pB.Asset2Name && pA.DEXName != pB.DEXName {
				graph.AddEdge(
					TradingPair{BaseToken: pA.Asset1Name, QuoteToken: pA.Asset2Name, DEXName: pA.DEXName},
					TradingPair{BaseToken: pB.Asset1Name, QuoteToken: pB.Asset2Name, DEXName: pB.DEXName},
					1.0, // TODO - ensure this adds 0 cost to transaction as the tokens are identical
				)
				graph.AddEdge(
					TradingPair{BaseToken: pB.Asset1Name, QuoteToken: pB.Asset2Name, DEXName: pB.DEXName},
					TradingPair{BaseToken: pA.Asset1Name, QuoteToken: pA.Asset2Name, DEXName: pA.DEXName},
					1.0, // TODO - same as above
				)
			}
		}
	}

	return graph
}

// NOTE - each pair in pars has the following function to execute a swap:
// ExecuteSwap(amountIn1, amountIn2 *big.Int) error 

// NOTE - *big.Int is used as 1 ETH will be represented as big.NewInt(1e18)
// Format:
// 
// type AmountOut struct {
//   Amount1 *big.Int // forward direction (1 asset0 -> ? asset1)
//   Amount2 *big.Int // backward direction (1 asset1 -> ? asset0)
// }
// 
// type SwapEvent struct {
//   DEXName    string
//   Asset1Name string
//  Asset2Name string
//  Address    string
//  AmountOut  AmountOut
// }

// Bellman-Ford algorithm
func detectArbitrageOpportunities(graph *Graph) {
	distances := make(map[TradingPair]float64)
	predecessors := make(map[TradingPair]*Edge)

	for node := range graph.Nodes {
		distances[node] = math.Inf(1)
	}

	var startNode TradingPair
	for node := range graph.Nodes {
		startNode = node
		break
	}

	distances[startNode] = 0

  // TODO - add more logging and checks to verify accuracy and give more data to console in real time

	for i := 0; i < len(graph.Nodes)-1; i++ {
		for node := range graph.Nodes {
			for _, edge := range graph.Edges[node] {
				if distances[node]+edge.Rate < distances[edge.To] {
					distances[edge.To] = distances[node] + edge.Rate
					predecessors[edge.To] = &edge
				}
			}
		}
	}

	for node := range graph.Nodes {
		for _, edge := range graph.Edges[node] {
			if distances[node]+edge.Rate < distances[edge.To] {
				log.Println("Arbitrage opportunity detected:")
				current := edge.To
				for {
					log.Printf("%s -> ", current)
					current = predecessors[current].From
					if current == edge.To || predecessors[current] == nil {
						break
					}
				}
				log.Printf("%s\n", edge.To)
				return
			}
		}
	}

	log.Println("No arbitrage opportunities found")
}

func Announce() {
	log.Printf("X-DEX X-Token Swap")
	log.Printf("==================")
}
