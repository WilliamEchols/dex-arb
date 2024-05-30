package types

import (
  "math/big"
  "context"
)

type Pair interface {
	Monitor(ctx context.Context, swapEventChan chan<- SwapEvent)
	ExecuteSwap(amountIn1, amountIn2 *big.Int) error
	Asset1() string
	Asset2() string
	DEX() string
	Address() string
}

type AmountOut struct {
  Amount1 *big.Float
  Amount2 *big.Float
}

type SwapEvent struct {
  DEXName    string
  Asset1Name string
  Asset2Name string
  Address    string
  AmountOut  AmountOut
}
