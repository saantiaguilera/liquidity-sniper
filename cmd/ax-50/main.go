package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	config2 "github.com/saantiaguilera/liquidity-ax-50/config"
	"github.com/saantiaguilera/liquidity-ax-50/pkg/domain"
	"github.com/saantiaguilera/liquidity-ax-50/pkg/service"
	services2 "github.com/saantiaguilera/liquidity-ax-50/pkg/services"
	erc202 "github.com/saantiaguilera/liquidity-ax-50/third_party/erc20"
	"github.com/saantiaguilera/liquidity-ax-50/third_party/pancake"
	"math/big"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Entry point of ax-50.
// Before Anything, check /config/*.json to correctly parametrize the bot

const (
	configFolderEnv = "CONF_DIR" // env var for setting the config folder. This is /config usually.
	configFile = "configurations"
)

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
	tokenIntance, _ := erc202.NewErc20(tokenAddress, client)
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
	if config2.BIG_BNB_TRANSFER == true {
		fmt.Println("activated\nthreshold of interest : transfers >", config2.BNB[:2], " BNB")
	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// ADDRESS MONITORING //////////////////\n")
	if config2.ADDRESS_MONITOR == true {
		fmt.Println("activated\nthe following addresses are monitored : \n")
		for addy, addressData := range config2.AddressesWatched {
			fmt.Println("address : ", addy, "name: ", addressData.Name)
		}
	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// LIQUIDITY SNIPING //////////////////\n")
	if config2.Sniping == true {
		fmt.Println("activated")
		name, _ := config2.Snipe.Tkn.Name(&bind.CallOpts{})
		fmt.Println("token targetted: ", config2.Snipe.TokenAddress, "(", name, ")")
		fmt.Println("minimum liquidity expected : ", formatEthWeiToEther(config2.Snipe.MinLiq), getTokenSymbol(config2.Snipe.TokenPaired, client))
		fmt.Println("current WBNB balance inside TRIGGER : ", formatEthWeiToEther(config2.GetTriggerWBNBBalance()), "WBNB")

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
	services2.TxClassifier(tx, client, TopSnipe)
}

func oldmain() {
	// we say <place_holder> for the defval as it is anyway filtered to geth_ipc in the switch
	ClientEntered := flag.String("client", "xxx", "Gateway to the bsc protocol. Available options:\n\t-bsc_testnet\n\t-bsc\n\t-geth_http\n\t-geth_ipc")
	flag.Parse()

	rpcClient := services2.InitRPCClient(ClientEntered)
	client := services2.GetCurrentClient() // NewClient(rpcClient)

	config2.InitDF(client)

	// init goroutine Clogg if config.Sniping == true
	if config2.Sniping == true {
		wg.Add(1)
		go func() {
			services2.Clogg(client, TopSnipe)
			wg.Done()
		}()
	}

	// Launch txpool streamer
	StreamNewTxs(client, rpcClient)

}

type (
	monitor interface {
		Monitor(context.Context, *types.Transaction)
	}
)

func main() {
	ctx := context.Background()

	conf, err := NewConfigFromFile(fmt.Sprintf("%s/%s.json", os.Getenv(configFolderEnv), configFile))
	if err != nil {
		panic(err) // halt immediately
	}

	log.Info(fmt.Sprintf("Configurations parsed: %+v\n", conf))

	rpcClient, err := rpc.DialContext(ctx, conf.Sniper.RPC)
	if err != nil {
		panic(err)
	}

	ethClient := ethclient.NewClient(rpcClient)

	chainID, err := ethClient.NetworkID(ctx)
	if err != nil {
		panic(err)
	}

	sniper := domain.NewSniper(
		conf.Sniper.Trigger,
		conf.Sniper.BaseCurrency,
		conf.Sniper.TargetToken,
		big.NewInt(int64(conf.Sniper.MinLiquidity)),
		chainID,
	)

	monitors := make([]monitor, 2)

	if conf.Monitors.AddressListMonitor.Enabled {
		l := make([]domain.NamedAddress, len(conf.Monitors.AddressListMonitor.List))
		for i, v := range conf.Monitors.AddressListMonitor.List {
			l[i] = domain.NewNamedAddress(v.Name, v.Addr)
		}

		monitors = append(monitors, service.NewAddressMonitor(sniper, l...))
	}

	if conf.Monitors.WhaleMonitor.Enabled {
		monitors = append(monitors, service.NewWhaleMonitor(big.NewInt(int64(conf.Monitors.WhaleMonitor.Min))))
	}

	cakeFactoryAddr, err := conf.Addr(AddressCakeFactory)
	if err != nil {
		panic(err)
	}

	pcsFactory, err := pancake.NewIPancakeFactory(cakeFactoryAddr, ethClient)
	if err != nil {
		panic(err)
	}

	sniperClient := service.NewSniper(ethClient, pcsFactory, ........)
} // TODO Repasar CAKE_ROUTER / etc que esten todos bien puestos