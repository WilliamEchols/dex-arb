package types

import (
  "math/big"
  "context"
)

type Pair interface {
  GetAmountOut() (*big.Int, *big.Int, error)
  Monitor(ctx context.Context, swapEventChan chan<- SwapEvent)
  ExecuteSwap(amountIn1, amountIn2 *big.Int) error 
}

type AmountOut struct {
  Amount1 *big.Int
  Amount2 *big.Int
}

type SwapEvent struct {
  DEXName    string
  Asset1Name string
  Asset2Name string
  Address    string
  AmountOut  AmountOut
}
