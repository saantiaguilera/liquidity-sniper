package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/saantiaguilera/liquidity-sniper/pkg/controller"
	"github.com/saantiaguilera/liquidity-sniper/pkg/service"
)

// Entry point of ax-50.
// Before Anything, check /config/*.json to correctly parametrize the bot

const (
	configFolderEnv     = "CONF_DIR" // env var for setting the config folder. This is /config usually.
	configFolderDefault = "config"
	configFile          = "local"
	beeBookFile         = "bee_book"

	logLevel = log.LvlDebug
)

func main() {
	configureLog(logLevel)
	ctx := context.Background()

	dir := os.Getenv(configFolderEnv)
	if len(dir) == 0 {
		dir = configFolderDefault
	}
	conf, err := NewConfigFromFile(fmt.Sprintf("%s/%s.json", dir, configFile))
	if err != nil {
		panic(err) // halt immediately
	}

	log.Info(fmt.Sprintf("Configurations parsed: %+v\n", conf))

	rpcClientRead := newRPCClient(ctx, conf.Chains.RChain.Node)
	rpcClientWrite := newRPCClient(ctx, conf.Chains.WChain.Node)
	ethClientRead := ethclient.NewClient(rpcClientRead)
	ethClientWrite := ethclient.NewClient(rpcClientWrite)

	chainIDRead, err := ethClientRead.NetworkID(ctx)
	if err != nil {
		panic(err)
	}
	chainIDWrite, err := ethClientWrite.NetworkID(ctx)
	if err != nil {
		panic(err)
	}

	if chainIDRead.Cmp(chainIDWrite) != 0 {
		panic("expected read and write clients on same chain id")
	}

	sniper := newSniperEntity(ctx, conf, ethClientWrite)
	monitors := newMonitors(conf, sniper)
	factory := newFactory(conf, ethClientWrite)
	swarm := newBees(ctx, ethClientWrite)
	monitorEngine := service.NewMonitorEngine(monitors...)
	sniperClient := service.NewSniper(ethClientWrite, factory, swarm, sniper)
	uniLiquidityClient := newUniswapLiquidityClient(ethClientWrite, sniperClient, sniper)

	txClassifierUseCase := newTxClassifierUseCase(conf, monitorEngine, uniLiquidityClient)

	txController := controller.NewTransaction(ethClientRead, txClassifierUseCase.Classify)

	log.Info("> Igniting engine.")
	NewEngine(rpcClientRead, txController).Run(ctx)
}
