package sushiswappair
// using same package as uniswapv2pair.go generated with abigen from UniswapV2Pair.abi

// TODO - document complete DEX implementation

import (
  "log"
  "fmt"
  "sync"
  "context"
  "math/big"
  "crypto/ecdsa"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

  "bb/types"
)

type Instance struct {
	AddressString      string
	Client             bind.ContractBackend
	PairInterface      *Uniswapv2pair
	auth               *bind.TransactOpts
	FeePerThousand     int64
	Asset1Name         string
	Asset2Name         string
	DEXName            string
  Asset1Decimals     int64
  Asset2Decimals     int64
}

func (i *Instance) Asset1() string {
	return i.Asset1Name
}

func (i *Instance) Asset2() string {
	return i.Asset2Name
}

func (i *Instance) DEX() string {
	return i.DEXName
}

func (i *Instance) Address() string {
	return i.AddressString
}

var wg sync.WaitGroup

func NewInstance(address string, client bind.ContractBackend, privateKey *ecdsa.PrivateKey, chainId *big.Int, asset1name string, asset2name string, asset1decimals int64, asset2decimals int64) *Instance {
  pair, err := NewUniswapv2pair(common.HexToAddress(address), client)
  if err != nil {
    log.Fatal(err)
  }

  auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
    log.Fatal(err)
	}

  return &Instance{
    AddressString:        address,
    Client:         client,
    PairInterface:  pair,
    auth:           auth,
    FeePerThousand: 3,
    Asset1Name:     asset1name,
    Asset2Name:     asset2name,
    DEXName:        "Sushiswap",
    Asset1Decimals: asset1decimals,
    Asset2Decimals: asset2decimals,
  }
}

func multiplyBy10PowX(value *big.Float, x int64) *big.Float {
    ten := big.NewInt(10)
    result := new(big.Float).Set(value) // Create a copy of the input value

    if x > 0 {
        // Positive exponent: Calculate 10^x using big.Int
        exponent := big.NewInt(x)
        tenToThePowerOfX := new(big.Int).Exp(ten, exponent, nil)
        floatTenToThePowerOfX := new(big.Float).SetInt(tenToThePowerOfX)
        result.Mul(result, floatTenToThePowerOfX)
    } else if x < 0 {
        // Negative exponent: Calculate 10^(-x) and then take the reciprocal
        exponent := big.NewInt(-x)
        tenToThePowerOfX := new(big.Int).Exp(ten, exponent, nil)
        floatTenToThePowerOfX := new(big.Float).SetInt(tenToThePowerOfX)
        result.Quo(result, floatTenToThePowerOfX)
    }
    // If x == 0, result is simply the original value, as 10^0 = 1

    return result
}

func (d *Instance) GetAmountOut() (*big.Float, *big.Float, error) {
	amountIn := new(big.Float).Mul(big.NewFloat(1), big.NewFloat(1)) // 1e18 should be 1 asset

	// Fetch reserves
	reserves, err := d.PairInterface.GetReserves(nil)
	if err != nil {
		return nil, nil, err
	}

	// Calculate amountOut for both directions
	amountOut1:= calculateAmountOut(amountIn, new(big.Float).SetInt(reserves.Reserve0), new(big.Float).SetInt(reserves.Reserve1))
  amountOut2 := new(big.Float).Quo(big.NewFloat(1), amountOut1)

  // Correct scaling for ETH/USDT
  amountOut1Scaled := multiplyBy10PowX(amountOut1, d.Asset1Decimals - d.Asset2Decimals - 3)
  amountOut2Scaled := multiplyBy10PowX(amountOut2, d.Asset2Decimals - d.Asset1Decimals - 3)

	return new(big.Float).Mul(amountOut1Scaled, big.NewFloat(float64(1000 - d.FeePerThousand))), new(big.Float).Mul(amountOut2Scaled, big.NewFloat(float64(1000 - d.FeePerThousand))), nil
}

func calculateAmountOut(amountIn, reserveIn, reserveOut *big.Float) *big.Float {
	numerator := new(big.Float).Mul(amountIn, reserveOut)
	denominator := new(big.Float).Add(new(big.Float).Mul(reserveIn, big.NewFloat(1)), amountIn)
	amountOut := new(big.Float).Quo(numerator, denominator)
	return amountOut
}

func (d *Instance) Monitor(ctx context.Context, swapEventChan chan<- types.SwapEvent) {
	defer wg.Done()

  swapChan := make(chan *Uniswapv2pairSwap)

  sub, err := d.PairInterface.WatchSwap(&bind.WatchOpts{Context: ctx}, swapChan, nil, nil)
  if err != nil {
    log.Fatal(err)
  }

	log.Printf("Listening for swap events: %s/%s on %s (%s)", d.Asset1Name, d.Asset2Name, d.DEXName, d.AddressString)

	for {
		select {
		case err := <-sub.Err():
			log.Fatalf("Subscription error: %v", err)
		case <-ctx.Done():
			return
		case swap := <-swapChan:
			_ = swap

			// Get amounts out
			forward, backward, err := d.GetAmountOut()
			if err != nil {
				log.Fatalf("GetAmountOut error: %v", err)
				return
			}

			amountOut := types.AmountOut{forward, backward}

			// Send update to channel
			swapEvent := types.SwapEvent{
				DEXName:    d.DEXName,
				Asset1Name: d.Asset1Name,
				Asset2Name: d.Asset2Name,
				Address:    d.AddressString,
				AmountOut:  amountOut,
			}
			swapEventChan <- swapEvent
		}
	}
}

func (d *Instance) ExecuteSwap(amountIn1, amountIn2 *big.Int) error {
  _, err := d.PairInterface.Swap(d.auth, amountIn1, amountIn2, common.Address{}, nil)
	if err != nil {
		return fmt.Errorf("failed to execute sell: %v", err)
	}

  // Assume success
  return nil
}
