package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/saantiaguilera/liquidity-sniper/pkg/domain"
	"github.com/saantiaguilera/liquidity-sniper/third_party/erc20"
)

const (
	nullHash = "0x0000000000000000000000000000000000000000000000000000000000000000"

	sniperMaxWaitTimeForTx = 20 * time.Second
)

var (
	triggerSmartContract = []byte{0x4e, 0xfa, 0xc3, 0x29} // function 'snipeListing' in our trigger smart contract.
	txValue              = big.NewInt(0)
	txGasLimit           = uint64(500000)
)

type (
	Sniper struct {
		mut *sync.Mutex

		factoryClient sniperFactoryClient // eg. PCS
		ethClient     sniperETHClient
		swarm         []*Bee

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
		RawPK        *ecdsa.PrivateKey
		PendingNonce uint64
	}

	txRes struct {
		Hash    common.Hash
		Receipt *types.Receipt
		Success bool
	}
)

func NewSniper(
	e sniperETHClient,
	f sniperFactoryClient,
	s []*Bee,
	sn domain.Sniper,
) *Sniper {

	return &Sniper{
		mut:               new(sync.Mutex),
		ethClient:         e,
		factoryClient:     f,
		swarm:             s,
		sniperTTBAddr:     common.HexToAddress(sn.AddressTargetToken),
		sniperTriggerAddr: common.HexToAddress(sn.AddressTrigger),
		sniperTokenPaired: common.HexToAddress(sn.AddressTargetPaired),
		sniperChainID:     sn.ChainID,
	}
}

func NewBee(
	rawPK *ecdsa.PrivateKey,
	pn uint64,
) *Bee {

	return &Bee{
		RawPK:        rawPK,
		PendingNonce: pn,
	}
}

// Snipe cloggs the mempool triggering our Trigger contract for performing the swap
//   gas provided will be used on all txs. It's ideal to use the same gas as the addLiq tx so our txs gets the same priority as the addLiq one
//
// Snipe is concurrently safe
func (c *Sniper) Snipe(ctx context.Context, gas *big.Int) error {
	c.mut.Lock()
	defer c.mut.Unlock()

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

	success := false
	for res := range finishedTxRes {
		if res.Success {
			success = true
			for _, l := range res.Receipt.Logs {
				if l.Address == c.sniperTTBAddr {
					hexAmount := hex.EncodeToString(l.Data)
					var value = new(big.Int)
					value.SetString(hexAmount, 16)
					var buf strings.Builder
					_, _ = buf.WriteString("Sniping succeeded!\n")
					_, _ = buf.WriteString(fmt.Sprintf("    Hash: %s\n", res.Hash.String()))
					_, _ = buf.WriteString(fmt.Sprintf("    Token: %s\n", c.sniperTTBAddr.String()))

					if amountBought, err := c.formatERC20Decimals(value, c.sniperTTBAddr); err == nil {
						_, _ = buf.WriteString(fmt.Sprintf("    Amount Bought: %.4f\n", amountBought))
					}

					if pairAddress, err := c.factoryClient.GetPair(&bind.CallOpts{}, c.sniperTTBAddr, c.sniperTokenPaired); err == nil {
						_, _ = buf.WriteString(fmt.Sprintf("    Pair Address: %s", pairAddress.String()))
					}

					log.Info(buf.String())
				}
			}
		}
	}
	if !success {
		log.Warn("Sniping failed. Check the sent transactions to see the reason (eg. minimum expected quantity of tokens couldn't be achieved)")
	}
	return nil
}

// Format # of tokens transferred into required float
func (c *Sniper) formatERC20Decimals(tokensSent *big.Int, tokenAddress common.Address) (float64, error) {
	// Create a ERC20 instance and connect to geth to get decimals
	tokenInstance, _ := erc20.NewErc20(tokenAddress, c.ethClient)
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
	return final, nil
}

// once all tx has been sent, check for status
func (c *Sniper) checkTxStatus(ctx context.Context, txHash common.Hash) txRes {
	if txHash == common.HexToHash(nullHash) {
		return txRes{
			Hash:    txHash,
			Success: false,
			Receipt: nil,
		}
	}

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

	s := time.Now()

	for range t.C {
		_, pend, err := c.ethClient.TransactionByHash(ctx, txHash)

		if !pend {
			break // see Stop() internals.
		}

		if err != nil {
			log.Error(fmt.Sprintf("error getting tx by hash %s: %s", txHash.String(), err))
		}

		// fail fast after waittime
		if time.Now().Add(-sniperMaxWaitTimeForTx).After(s) {
			return txRes{
				Hash:    txHash,
				Success: false,
				Receipt: nil,
			}
		}
	}

	receipt, err := c.ethClient.TransactionReceipt(ctx, txHash)

	if err != nil {
		log.Error(fmt.Sprintf("error getting tx receipt %s: %s", txHash.String(), err.Error()))
		return txRes{
			Hash:    txHash,
			Success: false,
			Receipt: nil,
		}
	}

	return txRes{
		Hash:    txHash,
		Success: receipt.Status == 1,
		Receipt: receipt,
	}
}

func (c *Sniper) execute(ctx context.Context, bee *Bee, gasPrice *big.Int) common.Hash {
	log.Debug(fmt.Sprintf("gas price using: %s", gasPrice.String()))
	nonce := bee.PendingNonce
	// create the tx
	txBee := types.NewTransaction(nonce, c.sniperTriggerAddr, txValue, txGasLimit, gasPrice, triggerSmartContract)
	// sign the tx
	signedTxBee, err := types.SignTx(txBee, types.LatestSignerForChainID(c.sniperChainID), bee.RawPK)
	if err != nil {
		log.Error(fmt.Sprintf("sendBee: problem with signedTxBee: %s", err))
		return common.HexToHash(nullHash)
	}

	err = c.ethClient.SendTransaction(ctx, signedTxBee)

	if err != nil {
		log.Error(fmt.Sprintf("error sending tx: %s", err.Error()))
		return common.HexToHash(nullHash)
	}
	log.Info(fmt.Sprintf("sent tx: %s", signedTxBee.Hash().Hex()))
	bee.PendingNonce++ // increment nonce for next one

	return signedTxBee.Hash()
}

func recovery() {
	if err := recover(); err != nil {
		log.Error(fmt.Sprintf("panic recovered: %s %s", fmt.Errorf("%s", err), debug.Stack()))
	}
}
