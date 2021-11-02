package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/saantiaguilera/liquidity-sniper/pkg/domain"
	"github.com/saantiaguilera/liquidity-sniper/pkg/service"
	"github.com/saantiaguilera/liquidity-sniper/third_party/uniswap"
)

type (
	bee struct {
		Address string `json:"addr"`
		PK      string `json:"pk"`
	}

	logHandler struct {
		format log.Format
	}
)

func (l *logHandler) Log(r *log.Record) error {
	fmt.Print(string(l.format.Format(r)))
	return nil
}

func configureLog(level log.Lvl) {
	glog := log.NewGlogHandler(&logHandler{
		format: log.TerminalFormat(true),
	})
	glog.Verbosity(level)
	log.Root().SetHandler(glog)
}

func newRPCClient(ctx context.Context, rpcURL string) *rpc.Client {
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		panic(err)
	}
	return rpcClient
}

func newSniperEntity(ctx context.Context, conf *Config, ethClient *service.EthClientCluster) domain.Sniper {
	chainID, err := ethClient.NetworkID(ctx)
	if err != nil {
		panic(err)
	}

	mul10pow14, _ := new(big.Int).SetString("100000000000000", 10)
	ml := big.NewInt(int64(10000 * conf.Sniper.MinLiquidity))
	ml.Mul(ml, mul10pow14)

	return domain.NewSniper(
		conf.Contracts.Trigger.Hex(),
		conf.Tokens.SnipeB.Hex(),
		conf.Tokens.SnipeA.Hex(),
		ml,
		chainID,
	)
}

func newMonitors(conf *Config, sniper domain.Sniper) []service.Monitor {
	monitors := make([]service.Monitor, 0, 2)

	if conf.Sniper.Monitors.AddressListMonitor.Enabled {
		l := make([]domain.NamedAddress, len(conf.Sniper.Monitors.AddressListMonitor.List))
		for i, v := range conf.Sniper.Monitors.AddressListMonitor.List {
			l[i] = domain.NewNamedAddress(v.Name, v.Addr.Hex())
		}

		monitors = append(monitors, service.NewAddressMonitor(sniper, l...).Monitor)
	}

	if conf.Sniper.Monitors.WhaleMonitor.Enabled {
		min, _ := new(big.Int).SetString(conf.Sniper.Monitors.WhaleMonitor.Min, 10)
		monitors = append(monitors, service.NewWhaleMonitor(min).Monitor)
	}

	return monitors
}

func newFactory(conf *Config, ethClient *service.EthClientCluster) *uniswap.IUniswapV2Factory {
	factoryAddr := conf.Contracts.Factory.Addr()
	factory, err := uniswap.NewIUniswapV2Factory(factoryAddr, ethClient)
	if err != nil {
		panic(err)
	}

	return factory
}

func newBees(ctx context.Context, ethClient *service.EthClientCluster) []*service.Bee {
	dir := os.Getenv(configFolderEnv)
	if len(dir) == 0 {
		dir = configFolderDefault
	}
	b, err := os.ReadFile(fmt.Sprintf("%s/%s.json", dir, beeBookFile))
	if err != nil {
		panic(err)
	}

	var swarm []bee
	err = json.Unmarshal(b, &swarm)
	if err != nil {
		panic(err)
	}

	res := make([]*service.Bee, len(swarm))
	for i, bee := range swarm {
		addr := common.HexToAddress(bee.Address)
		pn, err := ethClient.PendingNonceAt(ctx, addr)
		if err != nil {
			panic(err)
		}
		if pn == 0 {
			pn++
		}

		rawPK, err := crypto.HexToECDSA(bee.PK[2:])
		if err != nil {
			panic(err)
		}

		log.Info(fmt.Sprintf("creating bee %s with nonce: %d", bee.Address, pn))
		res[i] = service.NewBee(rawPK, pn)
	}

	return res
}

func newUniswapLiquidityClient(
	e *service.EthClientCluster,
	s *service.Sniper,
	sn domain.Sniper,
) *service.UniswapLiquidity {

	mul := txGasMultiplier
	if mul < 1 {
		mul = 1
	}
	v, err := service.NewUniswapLiquidity(e, s, sn, new(big.Float).SetFloat64(mul))
	if err != nil {
		panic(err)
	}
	return v
}
