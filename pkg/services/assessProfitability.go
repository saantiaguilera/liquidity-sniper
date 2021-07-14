package services

import (
	config2 "github.com/saantiaguilera/liquidity-ax-50/config"
	uniswap2 "github.com/saantiaguilera/liquidity-ax-50/third_party/uniswap"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Equivalent of _getAmountOut function of the PCS router. Calculates z.
func _getAmountOut(myMaxBuy, reserveOut, reserveIn *big.Int) *big.Int {

	var myMaxBuy9975 = new(big.Int)
	var z = new(big.Int)
	num := big.NewInt(9975)
	myMaxBuy9975.Mul(num, myMaxBuy)
	num.Mul(myMaxBuy9975, reserveOut)

	den := big.NewInt(10000)
	den.Mul(den, reserveIn)
	den.Add(den, myMaxBuy9975)
	z.Div(num, den)
	return z
}

// get reserves of a PCS pair an return it
func getReservesData(client *ethclient.Client) (*big.Int, *big.Int) {
	pairAddress, _ := config2.FACTORY.GetPair(&bind.CallOpts{}, SwapData.Token, config2.WBNB_ADDRESS)
	PAIR, _ := uniswap2.NewIPancakePair(pairAddress, client)
	reservesData, _ := PAIR.GetReserves(&bind.CallOpts{})
	if reservesData.Reserve0 == nil {
		return nil, nil
	}
	var Rtkn0 = new(big.Int)
	var Rbnb0 = new(big.Int)
	token0, _ := PAIR.Token0(&bind.CallOpts{})
	if token0 == config2.WBNB_ADDRESS {
		Rbnb0 = reservesData.Reserve0
		Rtkn0 = reservesData.Reserve1
	} else {
		Rbnb0 = reservesData.Reserve1
		Rtkn0 = reservesData.Reserve0
	}
	return Rtkn0, Rbnb0
}