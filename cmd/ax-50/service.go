package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/saantiaguilera/liquidity-sniper/pkg/domain"
	"github.com/saantiaguilera/liquidity-sniper/pkg/service"
	"github.com/saantiaguilera/liquidity-sniper/third_party/pancake"
)

type (
	bee struct {
		ID      int    `json:"id"`
		Address string `json:"address"`
		PK      string `json:"pk"`
	}
)

func newRPCClient(ctx context.Context, conf *Config) *rpc.Client {
	rpcClient, err := rpc.DialContext(ctx, conf.Sniper.RPC)
	if err != nil {
		panic(err)
	}
	return rpcClient
}

func newSniperEntity(ctx context.Context, conf *Config, ethClient *ethclient.Client) domain.Sniper {
	chainID, err := ethClient.NetworkID(ctx)
	if err != nil {
		panic(err)
	}

	mul10pow14, _ := new(big.Int).SetString("100000000000000", 10)
	ml := big.NewInt(int64(10000 * conf.Sniper.MinLiquidity))
	ml.Mul(ml, mul10pow14)

	return domain.NewSniper(
		conf.Sniper.Trigger,
		conf.Sniper.BaseCurrency,
		conf.Sniper.TargetToken,
		ml,
		chainID,
	)
}

func newMonitors(conf *Config, sniper domain.Sniper) []service.Monitor {
	monitors := make([]service.Monitor, 2)

	if conf.Monitors.AddressListMonitor.Enabled {
		l := make([]domain.NamedAddress, len(conf.Monitors.AddressListMonitor.List))
		for i, v := range conf.Monitors.AddressListMonitor.List {
			l[i] = domain.NewNamedAddress(v.Name, v.Addr)
		}

		monitors = append(monitors, service.NewAddressMonitor(sniper, l...).Monitor)
	}

	if conf.Monitors.WhaleMonitor.Enabled {
		min, _ := new(big.Int).SetString(conf.Monitors.WhaleMonitor.Min, 10)
		monitors = append(monitors, service.NewWhaleMonitor(min).Monitor)
	}

	return monitors
}

func newPCSFactory(conf *Config, ethClient *ethclient.Client) *pancake.IPancakeFactory {
	cakeFactoryAddr, err := conf.Addr(AddressCakeFactory)
	if err != nil {
		panic(err)
	}

	pcsFactory, err := pancake.NewIPancakeFactory(cakeFactoryAddr, ethClient)
	if err != nil {
		panic(err)
	}

	return pcsFactory
}

func newBees(ctx context.Context, ethClient *ethclient.Client) []*service.Bee {
	b, err := os.ReadFile(fmt.Sprintf("%s/%s.json", os.Getenv(configFolderEnv), beeBookFile))
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

		rawPK, err := crypto.HexToECDSA(bee.PK[2:])
		if err != nil {
			panic(err)
		}

		res[i] = service.NewBee(rawPK, pn)
	}

	return res
}

func newUniswapLiquidityClient(
	e *ethclient.Client,
	s *service.Sniper,
	sn domain.Sniper,
) *service.UniswapLiquidity {

	v, err := service.NewUniswapLiquidity(e, s, sn)
	if err != nil {
		panic(err)
	}
	return v
}
