package main

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/saantiaguilera/liquidity-sniper/pkg/controller"
	"github.com/saantiaguilera/liquidity-sniper/pkg/service"
	"github.com/saantiaguilera/liquidity-sniper/pkg/usecase"
)

// Entry point of ax-50.
// Before Anything, check /config/*.json to correctly parametrize the bot

const (
	configFolderEnv     = "CONF_DIR" // env var for setting the config folder. This is /config usually.
	configFolderDefault = "config"
	configFile          = "local"
	beeBookFile         = "bee_book"

	// workers is the number of concurrent jobs consuming events from the pool,
	// be careful not using something too low if the chain has high throughput for the specified mode
	// (else you will delay yourself because of your lack of processing power)
	// Using something too high will simply consume more resources on your end, try looking for the
	// sweet spot where you're not the bottleneck of the stream nor you are wasting resources.
	workers = 1000

	// txGasMultiplier when sniping non pending txs (eg. through new_blocks mode)
	// Since the add liquidity tx. is already confirmed there's no point in placing ourselves at the same priority
	// as the tx (because it has already been mined), hence we use more gas than the used for it to
	// increment our priority in the new block.
	//
	// It could be a static number too or we could estimate it but it would add latency to the bot, hence
	// we assume the add liquidity tx mined had already a good gas price + the multiplier increment pushing us.
	//
	// txGasMultiplier is required to be >=1. Eg
	//  - txGasMultiplier = 1: Same as the add liquidity tx gas price. No changes.
	//  - txGasMultiplier = 1.5: 50% more than the add liquidity tx gas price. eg. if tx had gasPrice 10, our txs will have 15.
	//  - txGasMultiplier = 0.5: Incorrect since it doesn't meet the constraints. Defaults to 1.
	//
	// This field is unused when the txs are already pending, thus when we are really frontrunning (eg. on pending_txs)
	txGasMultiplier = 1.25

	// logLevel of the logs. Using DEBUG/INFO may suffice,
	// if you want to check that everything works fine set LvlTrace (the lowest)
	logLevel = log.LvlInfo
)

func main() {
	if workers <= 0 {
		panic("workers > 0")
	}

	configureLog(logLevel)
	ctx := context.Background()

	dir := os.Getenv(configFolderEnv)
	if len(dir) == 0 {
		dir = configFolderDefault
	}
	conf, err := NewConfigFromFile(fmt.Sprintf("%s/%s.json", dir, configFile))
	if err != nil {
		panic(err)
	}

	log.Info(fmt.Sprintf("configurations parsed: %+v", conf))

	rpcClientStream := newRPCClient(ctx, conf.Chains.Nodes.Stream)

	/*
	* Currently we only allow one write client because:
	* - pending txs mode requires to use the same stream node, since gossip protocol may not notify another node yet
	*   and we would get race condition issues (because one node has info that another one hasn't yet)
	* - block mode isn't too resource consuming, meaning with a single node it should suffice
	*
	* Still, we leave the clustered load balancer in case in a future we need it. It shouldn't degrade performance
	* besides 2 function calls (that do nothing because they have early returns), so it's negligent.
	*
	* On the other hand, if the snipe and stream nodes are the same, we also reuse the connection because external nodes
	* may have clusters through an ELB which internally are.. also different nodes.
	* This doesn't matter much if it's a single instance self hosted node.
	**/
	var ecli *service.EthClientCluster
	if conf.Chains.Nodes.Snipe == conf.Chains.Nodes.Stream {
		ecli = service.NewEthClientCluster(ethclient.NewClient(rpcClientStream))
	} else {
		ecli = service.NewEthClientCluster(ethclient.NewClient(newRPCClient(ctx, conf.Chains.Nodes.Snipe)))
	}
	ctx = ecli.NewLoadBalancedContext(ctx)

	sniper := newSniperEntity(ctx, conf, ecli)
	monitors := newMonitors(conf, sniper)
	factory := newFactory(conf, ecli)
	swarm := newBees(ctx, ecli)
	monitorEngine := service.NewMonitorEngine(monitors...)
	sniperClient := service.NewSniper(ecli, factory, swarm, sniper)
	uniLiquidityClient := newUniswapLiquidityClient(ecli, sniperClient, sniper)

	txClassifierUseCase := newTxClassifierUseCase(conf, monitorEngine, uniLiquidityClient)

	log.Info("igniting engine")
	newEngine(conf, rpcClientStream, ecli, ecli.NewLoadBalancedContext, txClassifierUseCase).Run(ctx)
}

func newEngine(
	conf *Config,
	cli *rpc.Client,
	ecli *service.EthClientCluster,
	mid engineMid,
	uc *usecase.TransactionClassifier,
) *Engine {

	mode := conf.Sniper.Mode
	if len(mode) == 0 {
		mode = SniperModePendingTxs // defaults to pending txs
	}
	log.Info(fmt.Sprintf("using mode %s", mode))

	switch mode {
	case SniperModePendingTxs:
		ctrl := controller.NewPendingTransaction(ecli, uc.Classify)
		return NewEngine(
			cli,
			func(ctx context.Context, c *rpc.Client, ch chan<- interface{}) (*rpc.ClientSubscription, error) {
				return c.EthSubscribe(ctx, ch, "newPendingTransactions")
			},
			mid,
			func(ctx context.Context, v interface{}) error {
				return ctrl.Snipe(ctx, common.HexToHash(v.(string)))
			},
		)
	case SniperModeBlockScan:
		ctrl := controller.NewBlock(ecli, uc.Classify)
		return NewEngine(
			cli,
			func(ctx context.Context, c *rpc.Client, ch chan<- interface{}) (*rpc.ClientSubscription, error) {
				return c.EthSubscribe(ctx, ch, "newHeads")
			},
			mid,
			func(ctx context.Context, v interface{}) error {
				n := v.(map[string]interface{})["number"].(string)
				if bn, ok := new(big.Int).SetString(n[2:], 16); ok {
					return ctrl.Snipe(ctx, bn)
				}
				panic(fmt.Sprintf("%+v cannot be parsed to big int base 16 (hex)", n))
			},
		)
	default:
		panic(fmt.Sprintf("unknown sniper mode '%s'", mode))
	}
}
