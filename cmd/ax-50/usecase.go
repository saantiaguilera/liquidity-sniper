package main

import (
	"github.com/saantiaguilera/liquidity-AX-50/pkg/service"
	"github.com/saantiaguilera/liquidity-AX-50/pkg/usecase"
)

func newTxClassifierUseCase(
	conf *Config,
	monitorEngine *service.MonitorEngine,
	uniLiqClient *service.UniswapLiquidity,
) *usecase.TransactionClassifier {

	cakeRouterAddr, ok := conf.Addresses[string(AddressCakeRouter)]
	if !ok {
		panic("cake router address not found in config")
	}

	strats := make(map[[4]byte]usecase.TransactionClassifierStrategy)
	// Put the 4 bytes of each contract signature mapped to the strategy
	strats[[...]byte{0xf3, 0x05, 0xd7, 0x19}] = uniLiqClient.AddETH
	strats[[...]byte{0xe8, 0xe3, 0x37, 0x00}] = uniLiqClient.Add

	return usecase.NewTransactionClassifier(cakeRouterAddr, monitorEngine.Monitor, strats)
}
