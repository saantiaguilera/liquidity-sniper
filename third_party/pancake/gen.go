package pancake

//go:generate brew unlink solidity
//go:generate brew link solidity@6
//go:generate abigen --sol router.sol --pkg pancake --out router.go
