package main

import (
	"github.com/saantiaguilera/liquidity-sniper/pkg/service"
	"github.com/saantiaguilera/liquidity-sniper/pkg/usecase"
)

func newTxClassifierUseCase(
	conf *Config,
	monitorEngine *service.MonitorEngine,
	uniLiqClient *service.UniswapLiquidity,
) *usecase.TransactionClassifier {

	routerAddr := conf.Contracts.Router.Addr()
	strats := make(map[[4]byte]usecase.TransactionClassifierStrategy)
	// Put the 4 bytes of each contract signature mapped to the strategy
	strats[[...]byte{0xf3, 0x05, 0xd7, 0x19}] = uniLiqClient.AddETH
	strats[[...]byte{0xe8, 0xe3, 0x37, 0x00}] = uniLiqClient.Add

	return usecase.NewTransactionClassifier(routerAddr.Hex(), monitorEngine.Monitor, strats)
}
