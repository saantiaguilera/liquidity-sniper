package service

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/saantiaguilera/liquidity-ax-50/pkg/domain"
	"math/big"
)

const (
	addressMonitorGasMul = 1000000000
)

type (
	AddressMonitor struct {
		sniperChainID *big.Int
		watchedAddrs  map[common.Address]domain.NamedAddress
	}
)

func NewAddressMonitor(sn domain.Sniper, addrs ...domain.NamedAddress) *AddressMonitor {
	m := make(map[common.Address]domain.NamedAddress)
	for _, v := range addrs {
		m[common.HexToAddress(v.Addr)] = v
	}
	return &AddressMonitor{
		sniperChainID: sn.ChainID,
		watchedAddrs:  m,
	}
}

func (m *AddressMonitor) Monitor(ctx context.Context, tx *types.Transaction) {
	msg, err := tx.AsMessage(types.NewEIP155Signer(m.sniperChainID), nil)
	if err != nil {
		log.Error(err.Error())
		return
	}
	owner := msg.From()

	if na, ok := m.watchedAddrs[owner]; ok {
		log.Info(fmt.Sprintf(
`[AddressMonitor] New transaction from %s (%s)
    Nonce: %d
    GasPrice: %d
    Gas: %d
    Value: %d
    To: %s
    Hash: %s`,
			owner, na.Name, tx.Nonce(), formatETHWeiToEther(tx.GasPrice()), tx.Gas()*addressMonitorGasMul,
			formatETHWeiToEther(tx.Value()), tx.To(), tx.Hash(),
		))
	}
}