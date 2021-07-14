package uniswap

//go:generate brew unlink solidity
//go:generate brew link solidity@6
//go:generate abigen --sol router2.sol --pkg uniswap --out router.go
