package domain

import "math/big"

type (
	Sniper struct {
		// AddressTrigger of the smart contract
		AddressTrigger string
		// StandardGasPrice to use in txs
		StandardGasPrice big.Int
		// AddressBaseCurrency of the paired token. BNB's address probably.
		AddressBaseCurrency string
		// AddressTargetToken to liquidity snipe (buy)
		AddressTargetToken string
		// MinimumLiquidity expected when the dev adds liquidity.
		// We don't want to snipe if the team doesn't add a min amount of liquidity we expect.
		// It's an important question to solve in the telegram of the project.
		// You can also monitor bscscan and see the repartition of WBNB among the address that holds the targeted token
		// and deduce the WBNB liq that will be added.
		MinimumLiquidity int
	}
)
