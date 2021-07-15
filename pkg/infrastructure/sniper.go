package infrastructure

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/saantiaguilera/liquidity-ax-50/pkg/domain"
	erc202 "github.com/saantiaguilera/liquidity-ax-50/third_party/erc20"
	"math/big"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

var (
	triggerSmartContract = []byte{0x4e, 0xfa, 0xc3, 0x29}
	txValue              = big.NewInt(0)
	txGasLimit           = uint64(500000)
)

type (
	Sniper struct {
		park *int32

		factoryClient sniperFactoryClient // eg. PCS
		ethClient sniperETHClient
		swarm []*Bee

		sniperTTBAddr     common.Address
		sniperTriggerAddr common.Address
		sniperTokenPaired common.Address
		sniperChainID     *big.Int
	}

	sniperFactoryClient interface {
		GetPair(opts *bind.CallOpts, tokenA common.Address, tokenB common.Address) (common.Address, error)
	}

	sniperETHClient interface {
		bind.ContractBackend

		SendTransaction(context.Context, *types.Transaction) error

		TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
		TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	}

	Bee struct {
		ID           int
		Address      common.Address
		PK           string
		RawPK        *ecdsa.PrivateKey
		Balance      float64
		PendingNonce uint64
		GasPrice     big.Int
	}

	txRes struct {
		Hash   common.Hash
		Receipt *types.Receipt
		Success bool
	}
)

// TODO Here we use loadClogger() to create bees/swarm
func NewSniper(
	e sniperETHClient,
	f sniperFactoryClient,
	s []*Bee,
	sn domain.Sniper,
) *Sniper {

	p := int32(0)
	return &Sniper{
		park:              &p,
		ethClient:         e,
		factoryClient:     f,
		swarm:             s,
		sniperTTBAddr:     common.HexToAddress(sn.AddressTargetToken),
		sniperTriggerAddr: common.HexToAddress(sn.AddressTrigger),
		sniperTokenPaired: common.HexToAddress(sn.AddressBaseCurrency),
		sniperChainID:     sn.ChainID,
	}
}

func NewBee(
	id int,
	addr common.Address,
	pk string,
	rawPK *ecdsa.PrivateKey,
	balance float64,
	pn uint64,
	gp big.Int,
) *Bee {

	return &Bee{
		ID:           id,
		Address:      addr,
		PK:           pk,
		RawPK:        rawPK,
		Balance:      balance,
		PendingNonce: pn,
		GasPrice:     gp,
	}
}

func (c *Sniper) Snipe(ctx context.Context, gas *big.Int) error {
	if atomic.CompareAndSwapInt32(c.park, 0, 1) {
		defer atomic.SwapInt32(c.park, 0) // compensate always.

		wg := new(sync.WaitGroup)
		wg.Add(len(c.swarm))

		pendingTxRes := make(chan common.Hash, len(c.swarm))

		for _, b := range c.swarm {
			go func(ctx context.Context, b *Bee, wg *sync.WaitGroup, gas *big.Int, h chan<- common.Hash) {
				defer recovery()
				defer wg.Done()
				h <- c.execute(ctx, b, gas)
			}(ctx, b, wg, gas, pendingTxRes)
		}

		wg.Wait()
		close(pendingTxRes)

		log.Info(fmt.Sprintf("%d txs sent. Checking status...", len(c.swarm)))

		finishedTxRes := make(chan txRes, len(pendingTxRes))
		wg.Add(len(pendingTxRes))

		for txHash := range pendingTxRes {
			go func(ctx context.Context, h common.Hash, wg *sync.WaitGroup, ch chan<- txRes) {
				defer recovery()
				defer wg.Done()
				ch <- c.checkTxStatus(ctx, h)
			}(ctx, txHash, wg, finishedTxRes)
		}

		wg.Wait()
		close(finishedTxRes)

		for res := range finishedTxRes {
			if res.Success {
				// proudly displaying the tx receipt
				for _, l := range res.Receipt.Logs {
					if l.Address == c.sniperTTBAddr {
						hexAmount := hex.EncodeToString(l.Data)
						var value = new(big.Int)
						value.SetString(hexAmount, 16)
						amountBought, err := c.formatERC20Decimals(value, c.sniperTTBAddr)
						if err != nil {
							log.Info(fmt.Sprintf("sniping succeeded! but we couldn't get the bought balance: %s", err))
							continue
						}

						pairAddress, err := c.factoryClient.GetPair(&bind.CallOpts{}, c.sniperTTBAddr, c.sniperTokenPaired)
						if err != nil {
							log.Info(fmt.Sprintf("sniping succeeded! but we couldn't get the pair bought: %s", err))
							continue
						}

						log.Info(fmt.Sprintf(
							"sniping success!!!\nhash: %s\ntoken: %s\npairAddress: %s\namount bought: %.4f",
							res.Hash.String(),
							c.sniperTTBAddr.String(),
							pairAddress.String(),
							amountBought,
						))
					}
				}
			}
		}
	}
	return nil
}

// Format # of tokens transferred into required float
func (c *Sniper) formatERC20Decimals(tokensSent *big.Int, tokenAddress common.Address) (float64, error) {
	// Create a ERC20 instance and connect to geth to get decimals
	tokenInstance, _ := erc202.NewErc20(tokenAddress, c.ethClient)
	decimals, err := tokenInstance.Decimals(nil)
	if err != nil {
		return 0, err
	}
	// Construct a denominator based on the decimals
	// 18 decimals would result in denominator = 10^18
	var base, exponent = big.NewInt(10), big.NewInt(int64(decimals))
	denominator := base.Exp(base, exponent, nil)
	// Convert to float for precision
	tokensSentFloat := new(big.Float).SetInt(tokensSent)
	denominatorFloat := new(big.Float).SetInt(denominator)
	// Divide and return the final result
	final, _ := new(big.Float).Quo(tokensSentFloat, denominatorFloat).Float64()
	// TODO Take big.Accuracy into account
	return final, nil
}

// once all tx has been sent, check for status and feed StatusResults and WatchPending chan that are listening
func (c *Sniper) checkTxStatus(ctx context.Context, txHash common.Hash) txRes {

	t := time.NewTicker(500*time.Millisecond)
	defer t.Stop()

	s := time.Now()

	for range t.C {
		_, pend, err := c.ethClient.TransactionByHash(ctx, txHash)

		if !pend {
			break // see Stop() internals.
		}

		if err != nil {
			log.Info(err.Error())
		}

		// fail fast after 5s
		// TODO Use ctx?
		if time.Now().Add(-5*time.Second).After(s) {
			return txRes{
				Hash: txHash,
				Success: false,
				Receipt: nil,
			}
		}
	}

	receipt, err := c.ethClient.TransactionReceipt(ctx, txHash)

	if err != nil {
		log.Info(err.Error())
		return txRes{
			Hash: txHash,
			Success: false,
			Receipt: nil,
		}
	}

	return txRes{
		Hash:   txHash,
		Success: receipt.Status == 1,
		Receipt: receipt,
	}
}

func (c *Sniper) execute(ctx context.Context, bee *Bee, gasPrice *big.Int) common.Hash {
	nonce := bee.PendingNonce
	// create the tx
	txBee := types.NewTransaction(nonce, c.sniperTriggerAddr, txValue, txGasLimit, gasPrice, triggerSmartContract)
	// sign the tx
	signedTxBee, err := types.SignTx(txBee, types.NewEIP155Signer(c.sniperChainID), bee.RawPK)
	if err != nil {
		log.Info(fmt.Sprintf("sendBee: problem with signedTxBee: %s", err))
	}

	// send the tx
	// TODO Ctx timeout?
	err = c.ethClient.SendTransaction(ctx, signedTxBee)

	if err != nil {
		log.Info(fmt.Sprintf("error sending tx: %s", err.Error()))
		return common.HexToHash(domain.NullHash)
	}
	log.Info(fmt.Sprintf("sent tx: %s", signedTxBee.Hash().Hex()))
	return signedTxBee.Hash()
}

func recovery() {
	if err := recover(); err != nil {
		log.Error(fmt.Sprintf("panic recovered: %s %s", fmt.Errorf("%s", err), debug.Stack()))
	}
}