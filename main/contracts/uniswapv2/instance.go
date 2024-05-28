package uniswapv2pair
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

  "bb-e/types"
)

type Instance struct {
  Address string
  Client bind.ContractBackend
  PairInterface *Uniswapv2pair
  auth *bind.TransactOpts // auth is private
  FeePerThousand int64
  Asset1Name string
  Asset2Name string
  DEXName string
}

var wg sync.WaitGroup

func NewInstance(address string, client bind.ContractBackend, privateKey *ecdsa.PrivateKey, chainId *big.Int, asset1name string, asset2name string) *Instance {
  pair, err := NewUniswapv2pair(common.HexToAddress(address), client)
  if err != nil {
    log.Fatal(err)
  }

  auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
    log.Fatal(err)
	}

  return &Instance{
    Address:        address,
    Client:         client,
    PairInterface:  pair,
    auth:           auth,
    FeePerThousand: 3,
    Asset1Name:     asset1name,
    Asset2Name:     asset2name,
    DEXName:        "UniswapV2",
  }
}

// Returns forward (1->2), backward (2->1), and error
func (d *Instance) GetAmountOut() (*big.Int, *big.Int, error) {
  reserves, err := d.PairInterface.GetReserves(nil)
  if err != nil {
    return nil, nil, err
  }

  amountIn := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))

  reserveIn := reserves.Reserve0
  reserveOut := reserves.Reserve1

  amountInWithFee := new(big.Int).Mul(amountIn, big.NewInt(1000 - d.FeePerThousand))
  numerator := new(big.Int).Mul(amountInWithFee, reserveOut)
  denominator := new(big.Int).Add(new(big.Int).Mul(reserveIn, big.NewInt(1000)), amountInWithFee)
  forward := new(big.Int).Div(numerator, denominator)

  reserveIn = reserves.Reserve1
  reserveOut = reserves.Reserve0

  numerator = new(big.Int).Mul(amountInWithFee, reserveOut)
  denominator = new(big.Int).Add(new(big.Int).Mul(reserveIn, big.NewInt(1000)), amountInWithFee)
  backward := new(big.Int).Div(numerator, denominator)

  return forward, backward, nil
}

func (d *Instance) Monitor(ctx context.Context, swapEventChan chan<- types.SwapEvent) {
  defer wg.Done()

  swapChan := make(chan *Uniswapv2pairSwap)

  sub, err := d.PairInterface.WatchSwap(&bind.WatchOpts{Context: ctx}, swapChan, nil, nil)
  if err != nil {
    log.Fatal(err)
  }

  log.Printf("Subscribed to swap events for %s/%s on %s (%s)", d.Asset1Name, d.Asset2Name, d.DEXName, d.Address)

  for {
    select {
	    case err := <-sub.Err():
			  log.Fatalf("Subscription error: %v", err)
      case <-ctx.Done():
        return
      case swap := <-swapChan:
        _ = swap

        // Fetch reserves to calculate amountOut
        reserves, err := d.PairInterface.GetReserves(nil)
        if err != nil {
          log.Fatalf("GetReserves in Monitor error: %v", err)
          return 
        }

        amountIn := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))

        reserveIn := reserves.Reserve0
        reserveOut := reserves.Reserve1

        amountInWithFee := new(big.Int).Mul(amountIn, big.NewInt(1000 - d.FeePerThousand))
        numerator := new(big.Int).Mul(amountInWithFee, reserveOut)
        denominator := new(big.Int).Add(new(big.Int).Mul(reserveIn, big.NewInt(1000)), amountInWithFee)
        forward := new(big.Int).Div(numerator, denominator)

        reserveIn = reserves.Reserve1
        reserveOut = reserves.Reserve0

        numerator = new(big.Int).Mul(amountInWithFee, reserveOut)
        denominator = new(big.Int).Add(new(big.Int).Mul(reserveIn, big.NewInt(1000)), amountInWithFee)
        backward := new(big.Int).Div(numerator, denominator)

        amountOut := types.AmountOut{forward, backward}

        swapEvent := types.SwapEvent{
          DEXName:       d.DEXName,
          Asset1Name:    d.Asset1Name,
          Asset2Name:    d.Asset2Name,
          Address:       d.Address,
          AmountOut:     amountOut,
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
