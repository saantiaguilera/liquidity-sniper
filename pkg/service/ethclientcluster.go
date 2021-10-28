package service

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type (
	EthClientCluster struct {
		delegates []EthClient

		count uint
		mut   *sync.Mutex
	}

	EthClient interface {
		bind.ContractBackend

		SendTransaction(context.Context, *types.Transaction) error

		TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
		TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)

		BlockByNumber(context.Context, *big.Int) (b *types.Block, err error)

		NetworkID(context.Context) (*big.Int, error)
	}

	ethClientClusterCtxKey struct{}
)

func NewEthClientCluster(d ...EthClient) *EthClientCluster {
	return &EthClientCluster{
		delegates: d,
		count:     0,
		mut:       new(sync.Mutex),
	}
}

func (e *EthClientCluster) NewLoadBalancedContext(ctx context.Context) context.Context {
	if len(e.delegates) == 1 { // early return if no need to decorate.
		return ctx
	}

	e.mut.Lock()
	defer e.mut.Unlock()
	e.count++ // overflow handled transparently
	return context.WithValue(ctx, ethClientClusterCtxKey{}, e.count)
}

func (e *EthClientCluster) delegateAt(ctx context.Context) EthClient {
	if len(e.delegates) == 1 { // just one, go fast.
		return e.delegates[0]
	}

	n, ok := ctx.Value(ethClientClusterCtxKey{}).(uint)
	if !ok {
		log.Warn("trying to use an eth client without providing a load balanced context")
		n = 0
	}
	return e.delegates[(int(n % uint(len(e.delegates))))]
}

func (e *EthClientCluster) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return e.delegateAt(ctx).CodeAt(ctx, contract, blockNumber)
}

func (e *EthClientCluster) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return e.delegateAt(ctx).CallContract(ctx, call, blockNumber)
}

func (e *EthClientCluster) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return e.delegateAt(ctx).HeaderByNumber(ctx, number)
}

func (e *EthClientCluster) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return e.delegateAt(ctx).PendingCodeAt(ctx, account)
}

func (e *EthClientCluster) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return e.delegateAt(ctx).PendingNonceAt(ctx, account)
}

func (e *EthClientCluster) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return e.delegateAt(ctx).SuggestGasPrice(ctx)
}

func (e *EthClientCluster) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return e.delegateAt(ctx).SuggestGasTipCap(ctx)
}

func (e *EthClientCluster) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	return e.delegateAt(ctx).EstimateGas(ctx, call)
}

func (e *EthClientCluster) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return e.delegateAt(ctx).SendTransaction(ctx, tx)
}

func (e *EthClientCluster) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return e.delegateAt(ctx).FilterLogs(ctx, query)
}

func (e *EthClientCluster) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return e.delegateAt(ctx).SubscribeFilterLogs(ctx, query, ch)
}

func (e *EthClientCluster) TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	return e.delegateAt(ctx).TransactionByHash(ctx, hash)
}

func (e *EthClientCluster) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return e.delegateAt(ctx).TransactionReceipt(ctx, txHash)
}

func (e *EthClientCluster) NetworkID(ctx context.Context) (*big.Int, error) {
	return e.delegateAt(ctx).NetworkID(ctx)
}

func (e *EthClientCluster) BlockByNumber(ctx context.Context, n *big.Int) (b *types.Block, err error) {
	return e.delegateAt(ctx).BlockByNumber(ctx, n)
}
