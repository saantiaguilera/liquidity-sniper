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
	configFolderEnv = "CONF_DIR" // env var for setting the config folder. This is /config usually.
	configFile      = "configurations"
	beeBookFile     = "bee_book"
)

func main() {
	ctx := context.Background()

	conf, err := NewConfigFromFile(fmt.Sprintf("%s/%s.json", os.Getenv(configFolderEnv), configFile))
	if err != nil {
		panic(err) // halt immediately
	}

	log.Info(fmt.Sprintf("Configurations parsed: %+v\n", conf))

	rpcClient := newRPCClient(ctx, conf)
	ethClient := ethclient.NewClient(rpcClient)
	sniper := newSniperEntity(ctx, conf, ethClient)
	monitors := newMonitors(conf, sniper)
	pcsFactory := newPCSFactory(conf, ethClient)
	swarm := newBees(ctx, ethClient)
	monitorEngine := service.NewMonitorEngine(monitors...)
	sniperClient := service.NewSniper(ethClient, pcsFactory, swarm, sniper)
	uniLiquidityClient := newUniswapLiquidityClient(ethClient, sniperClient, sniper)

	txClassifierUseCase := newTxClassifierUseCase(conf, monitorEngine, uniLiquidityClient)

	txController := controller.NewTransaction(ethClient, txClassifierUseCase.Classify)

	NewEngine(rpcClient, txController).Run(ctx)
}
