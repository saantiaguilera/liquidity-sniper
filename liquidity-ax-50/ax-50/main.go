package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"sync"

	"github.com/saantiaguilera/liquidity-AX-50/ax-50/config"
	"github.com/saantiaguilera/liquidity-AX-50/ax-50/contracts/erc20"
	"github.com/saantiaguilera/liquidity-AX-50/ax-50/services"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Entry point of ax-50.
// Before Anything, check /config/config to correctly parametrize the bot

var wg = sync.WaitGroup{}
var TopSnipe = make(chan *big.Int)

// convert WEI to ETH
func formatEthWeiToEther(etherAmount *big.Int) float64 {
	var base, exponent = big.NewInt(10), big.NewInt(18)
	denominator := base.Exp(base, exponent, nil)
	tokensSentFloat := new(big.Float).SetInt(etherAmount)
	denominatorFloat := new(big.Float).SetInt(denominator)
	final, _ := new(big.Float).Quo(tokensSentFloat, denominatorFloat).Float64()
	return final
}

// fetch ERC20 token symbol
func getTokenSymbol(tokenAddress common.Address, client *ethclient.Client) string {
	tokenIntance, _ := erc20.NewErc20(tokenAddress, client)
	sym, _ := tokenIntance.Symbol(nil)
	return sym
}

// main loop of the bot
func StreamNewTxs(client *ethclient.Client, rpcClient *rpc.Client) {

	// Go channel to pipe data from client subscription
	newTxsChannel := make(chan common.Hash)

	// Subscribe to receive one time events for new txs
	_, err := rpcClient.EthSubscribe(
		context.Background(), newTxsChannel, "newPendingTransactions", // no additional args
	)

	if err != nil {
		fmt.Println("error while subscribing: ", err)
	}
	fmt.Println("\nSubscribed to mempool txs!\n")

	fmt.Println("\n////////////// BIG TRANSFERS //////////////////\n")
	if config.BIG_BNB_TRANSFER == true {
		fmt.Println("activated\nthreshold of interest : transfers >", config.BNB[:2], " BNB")
	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// ADDRESS MONITORING //////////////////\n")
	if config.ADDRESS_MONITOR == true {
		fmt.Println("activated\nthe following addresses are monitored : \n")
		for addy, addressData := range config.AddressesWatched {
			fmt.Println("address : ", addy, "name: ", addressData.Name)
		}
	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// SANDWICHER //////////////////\n")
	if config.Sandwicher == true {
		fmt.Println("activated\n\nmax BNB amount authorised for one sandwich : ", config.Sandwicher_maxbound, "WBNB")
		fmt.Println("minimum profit expected : ", config.Sandwicher_minprofit, "WBNB")
		fmt.Println("current WBNB balance inside TRIGGER : ", formatEthWeiToEther(config.GetTriggerWBNBBalance()), "WBNB")
		fmt.Println("TRIGGER balance at which we stop execution : ", formatEthWeiToEther(config.STOPLOSSBALANCE), "WBNB")
		fmt.Println("WARNING: be sure TRIGGER WBNB balance is > SANDWICHER MAXBOUND !!")

		activeMarkets := 0
		for _, specs := range config.SANDWICH_BOOK {
			if specs.Whitelisted == true && specs.ManuallyDisabled == false {
				// fmt.Println(specs.Name, market, specs.Liquidity)
				activeMarkets += 1
			}
		}
		fmt.Println("\nNumber of active Markets: ", activeMarkets, "\n")

		fmt.Println("\nManually disabled Markets: \n")
		for market, specs := range config.SANDWICH_BOOK {
			if specs.ManuallyDisabled == true {
				fmt.Println(specs.Name, market, specs.Liquidity)
			}
		}
		fmt.Println("\nEnnemies: \n")
		for ennemy := range config.ENNEMIES {
			fmt.Println(ennemy)
		}

	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// LIQUIDITY SNIPING //////////////////\n")
	if config.Sniping == true {
		fmt.Println("activated")
		name, _ := config.Snipe.Tkn.Name(&bind.CallOpts{})
		fmt.Println("token targetted: ", config.Snipe.TokenAddress, "(", name, ")")
		fmt.Println("minimum liquidity expected : ", formatEthWeiToEther(config.Snipe.MinLiq), getTokenSymbol(config.Snipe.TokenPaired, client))
		fmt.Println("current WBNB balance inside TRIGGER : ", formatEthWeiToEther(config.GetTriggerWBNBBalance()), "WBNB")

	} else {
		fmt.Println("not activated")
	}
	chainID, _ := client.NetworkID(context.Background())
	signer := types.NewEIP155Signer(chainID)

	for {
		select {
		// Code block is executed when a new tx hash is piped to the channel
		case transactionHash := <-newTxsChannel:
			// Get transaction object from hash by querying the client
			tx, is_pending, _ := client.TransactionByHash(context.Background(), transactionHash)
			// If tx is valid and still unconfirmed
			if is_pending {
				_, _ = signer.Sender(tx)
				handleTransaction(tx, client)
			}
		}
	}
}

func handleTransaction(tx *types.Transaction, client *ethclient.Client) {
	services.TxClassifier(tx, client, TopSnipe)
}

func main() {

	// we say <place_holder> for the defval as it is anyway filtered to geth_ipc in the switch
	ClientEntered := flag.String("client", "xxx", "Gateway to the bsc protocol. Available options:\n\t-bsc_testnet\n\t-bsc\n\t-geth_http\n\t-geth_ipc")
	flag.Parse()

	rpcClient := services.InitRPCClient(ClientEntered)
	client := services.GetCurrentClient()

	config.InitDF(client)

	// init goroutine Clogg if config.Sniping == true
	if config.Sniping == true {
		wg.Add(1)
		go func() {
			services.Clogg(client, TopSnipe)
			wg.Done()
		}()
	}

	// Launch txpool streamer
	StreamNewTxs(client, rpcClient)

}
