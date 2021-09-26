package uniswap

//go:generate brew unlink solidity@6
//go:generate brew link solidity
//go:generate abigen --abi UniswapV2Router02.abi --pkg uniswap --out router.go
