# dex-arb 

Cross-exchange and cross-asset-pairs arbitrage of the cryptocurrency market using decentralized exchanges (DEXs). 

# Metholodigy

Assets are treated as nodes in a graph with available swaps as traversable edges. Complete k-cycles identify trade opportunities with cumulative weight indicating trade PnL.

# Language

Written in Golang to utilize go-routines for each websocket to listen to each exchange live.
