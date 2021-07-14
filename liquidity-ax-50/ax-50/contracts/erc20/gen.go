package erc20

//go:generate brew unlink solidity@6
//go:generate brew link solidity
//go:generate abigen --abi ERC20.abi --pkg erc20 --out erc20.go
