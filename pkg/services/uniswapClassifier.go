package services

import (
	"encoding/json"
	"fmt"
	config2 "github.com/saantiaguilera/liquidity-ax-50/config"
	uniswap2 "github.com/saantiaguilera/liquidity-ax-50/third_party/uniswap"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// listing of pcs/uni function selectors of interest. We use the bytes version as we want to be super fast
var swapExactETHForTokens = [4]byte{0x7f, 0xf3, 0x6a, 0xb5}
var swapExactTokensForETH = [4]byte{0x18, 0xcb, 0xaf, 0xe5}
var swapExactTokensForTokens = [4]byte{0x38, 0xed, 0x17, 0x39}
var swapETHForExactTokens = [4]byte{0xfb, 0x3b, 0xdb, 0x41}
var swapTokensForExactETH = [4]byte{0x4a, 0x25, 0xd9, 0x4a}
var swapTokensForExactTokens = [4]byte{0x88, 0x03, 0xdb, 0xee}
var addLiquidityETH = [4]byte{0xf3, 0x05, 0xd7, 0x19}
var addLiquidity = [4]byte{0xe8, 0xe3, 0x37, 0x00}

// standard ABI
var routerAbi, _ = abi.JSON(strings.NewReader(uniswap2.PancakeRouterABI))

// interest Sniping and filter addliquidity tx
func HandleAddLiquidity(tx *types.Transaction, client *ethclient.Client, topSnipe chan *big.Int) {

	// parse the info of the swap so that we can access it easily
	var addLiquidity = buildAddLiquidityData(tx)

	sender := getTxSenderAddressQuick(tx, client)
	// security checks
	// does the liquidity addition deals with the token i'm targetting?
	if addLiquidity.TokenAddressA == config2.Snipe.TokenAddress || addLiquidity.TokenAddressB == config2.Snipe.TokenAddress {
		// does the liquidity is added on the right pair?
		if addLiquidity.TokenAddressA == config2.Snipe.TokenPaired || addLiquidity.TokenAddressB == config2.Snipe.TokenPaired {
			tknBalanceSender, _ := config2.Snipe.Tkn.BalanceOf(&bind.CallOpts{}, sender)
			var AmountTknMin *big.Int
			var AmountPairedMin *big.Int
			if addLiquidity.TokenAddressA == config2.Snipe.TokenAddress {
				AmountTknMin = addLiquidity.AmountTokenAMin
				AmountPairedMin = addLiquidity.AmountTokenBMin
			} else {
				AmountTknMin = addLiquidity.AmountTokenBMin
				AmountPairedMin = addLiquidity.AmountTokenAMin
			}
			// we check if the liquidity provider really possess the liquidity he wants to add, because it is possible tu be lured by other bots that fake liquidity addition.
			checkBalanceTknLP := AmountTknMin.Cmp(tknBalanceSender)
			if checkBalanceTknLP == 0 || checkBalanceTknLP == -1 {
				// we check if the liquidity provider add enough collateral (WBNB or BUSD) as expected by our configuration. Bc sometimes the dev fuck the pleb and add way less liquidity that was advertised on telegram.
				if AmountPairedMin.Cmp(config2.Snipe.MinLiq) == 1 {
					if SNIPEBLOCK == false {

						// reminder: the Clogg goroutine launched in ax-50.go is still blocking and is waiting for the gas price value. Here we unblock it. And all the armed bees are launched, which clogg the mempool and increase the chances of successful sniping.
						topSnipe <- tx.GasPrice()

						// following is just verbose / design thing
						var final = buildAddLiquidityFinal(tx, client, &addLiquidity)
						out, _ := json.MarshalIndent(final, "", "\t")
						fmt.Println("PankakeSwap: New Liquidity addition:")
						fmt.Println(string(out))
					} else {
						fmt.Println("SNIPEBLOCK activated. Must relaunch Clogger to perform another snipe")
					}

				} else {
					fmt.Println("liquidity added but lower than expected : ", formatEthWeiToEther(AmountPairedMin), getTokenSymbol(config2.Snipe.TokenPaired, client), " vs", formatEthWeiToEther(config2.Snipe.MinLiq), " expected")
				}
			}
		}
	}
}

// interest Sniping and filter addliquidity tx
func HandleAddLiquidityETH(tx *types.Transaction, client *ethclient.Client, topSnipe chan *big.Int) {
	// parse the info of the swap so that we can access it easily
	var addLiquidity = buildAddLiquidityEthData(tx)
	sender := getTxSenderAddressQuick(tx, client)
	tknBalanceSender, _ := config2.Snipe.Tkn.BalanceOf(&bind.CallOpts{}, sender)
	checkBalanceLP := addLiquidity.AmountTokenMin.Cmp(tknBalanceSender)

	// security checks:
	// does the liquidity addition deals with the token i'm targetting?
	if addLiquidity.TokenAddress == config2.Snipe.TokenAddress {
		// we check if the liquidity provider really possess the liquidity he wants to add, because it is possible tu be lured by other bots that fake liquidity addition.
		if checkBalanceLP == 0 || checkBalanceLP == -1 {
			// we check if the liquidity provider add enough collateral (WBNB or BUSD) as expected by our configuration. Bc sometimes the dev fuck the pleb and add way less liquidity that was advertised on telegram.
			if tx.Value().Cmp(config2.Snipe.MinLiq) == 1 {
				if addLiquidity.AmountETHMin.Cmp(config2.Snipe.MinLiq) == 1 {
					if SNIPEBLOCK == false {
						// reminder: the Clogg goroutine launched in ax-50.go is still blocking and is waiting for the gas price value. Here we unblock it. And all the armed bees are launched, which clogg the mempool and increase the chances of successful sniping.
						topSnipe <- tx.GasPrice()

						// following is just verbose / design thing
						var final = buildAddLiquidityEthFinal(tx, client, &addLiquidity)
						out, _ := json.MarshalIndent(final, "", "\t")
						fmt.Println("PankakeSwap: New BNB Liquidity addition:")
						fmt.Println(string(out))
					} else {
						fmt.Println("SNIPEBLOCK activated. Must relaunch Clogger to perform another snipe")
					}
				}
			} else {
				fmt.Println("liquidity added but lower than expected : ", formatEthWeiToEther(tx.Value()), " BNB", " vs", formatEthWeiToEther(config2.Snipe.MinLiq), " expected")
			}
		}
	}
}

// Core method that determines the kind of uniswap trade the tx is
func handleUniswapTrade(tx *types.Transaction, client *ethclient.Client, topSnipe chan *big.Int) {

	UNISWAPBLOCK = true
	txFunctionHash := [4]byte{}
	copy(txFunctionHash[:], tx.Data()[:4])
	switch txFunctionHash {

	case addLiquidityETH:
		if config2.PCS_ADDLIQ == true {
			HandleAddLiquidityETH(tx, client, topSnipe)
		}
	case addLiquidity:
		if config2.PCS_ADDLIQ == true {
			HandleAddLiquidity(tx, client, topSnipe)
		}
	}
	UNISWAPBLOCK = false
}
