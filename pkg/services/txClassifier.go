package services

import (
	"fmt"
	config2 "github.com/saantiaguilera/liquidity-ax-50/config"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// allow atomic treatment of Pancakeswap pending tx
var UNISWAPBLOCK = false

// sniping is considered as a one time event. Lock the fuctionality once a snipe occured
var SNIPEBLOCK = true

// Core classifier to tag txs in the mempool before they're executed. Only used for PCS tx for now but other filters could be added
func TxClassifier(tx *types.Transaction, client *ethclient.Client, topSnipe chan *big.Int) {
	// fmt.Println("new tx to TxClassifier")
	if tx.To() != nil {
		if config2.AddressesWatched[getTxSenderAddressQuick(tx, client)].Watched == true {
			go handleWatchedAddressTx(tx, client)
		} else if tx.To().Hex() == config2.CAKE_ROUTER_ADDRESS {
			if UNISWAPBLOCK == false && len(tx.Data()) >= 4 {
				// pankakeSwap events are managed in their own file uniswapClassifier.go
				go handleUniswapTrade(tx, client, topSnipe)
			}
		} else if tx.Value().Cmp(&config2.BigTransfer) == 1 && config2.BIG_BNB_TRANSFER == true {
			fmt.Printf("\nBIG TRANSFER: %v, Value: %v\n", tx.Hash().Hex(), formatEthWeiToEther(tx.Value()))
		}
	}
}

// display transactions of the address you monitor if ADDRESS_MONITOR == true in the config file
func handleWatchedAddressTx(tx *types.Transaction, client *ethclient.Client) {
	sender := getTxSenderAddressQuick(tx, client)
	fmt.Println("New transaction from ", sender, "(", config2.AddressesWatched[sender].Name, ")")
	fmt.Println("Nonce : ", tx.Nonce())
	fmt.Println("GasPrice : ", formatEthWeiToEther(tx.GasPrice()))
	fmt.Println("Gas : ", tx.Gas()*1000000000)
	fmt.Println("Value : ", formatEthWeiToEther(tx.Value()))
	fmt.Println("To : ", tx.To())
	fmt.Println("Hash : ", tx.Hash(), "\n")
}
