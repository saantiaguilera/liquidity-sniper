package service

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/saantiaguilera/liquidity-sniper/pkg/domain"
	"github.com/saantiaguilera/liquidity-sniper/third_party/erc20"
)

var (
	minGasMultiplier = new(big.Float).SetInt64(1) // minimum gas multiplier. Should at least be 1, else it's lower than the provided.
)

type (
	UniswapLiquidity struct {
		ethClient    uniswapLiquidityETHClient
		sniperClient uniswapLiquiditySniperClient

		sniperTTBAddr     common.Address
		sniperTTBTkn      *erc20.Erc20
		sniperTokenPaired common.Address
		sniperMinLiq      *big.Int
		sniperChainID     *big.Int

		gasMultiplier *big.Float // useful when the tx is already confirmed and we are not really frontrunning per-se.
	}

	uniswapLiquidityETHClient interface {
		bind.ContractBackend

		NetworkID(context.Context) (*big.Int, error)
	}

	uniswapLiquiditySniperClient interface {
		Snipe(context.Context, *big.Int) error
	}

	uniswapAddLiquidityInput struct {
		TokenAddressA       common.Address
		TokenAddressB       common.Address
		AmountTokenADesired *big.Int
		AmountTokenBDesired *big.Int
		AmountTokenAMin     *big.Int
		AmountTokenBMin     *big.Int
		Deadline            *big.Int
		To                  common.Address
	}

	uniswapAddLiquidityETHInput struct {
		TokenAddress       common.Address
		AmountTokenDesired *big.Int
		AmountTokenMin     *big.Int
		AmountETHMin       *big.Int
		Deadline           *big.Int
		To                 common.Address
	}
)

func NewUniswapLiquidity(
	e uniswapLiquidityETHClient,
	s uniswapLiquiditySniperClient,
	sn domain.Sniper,
	gasMultiplier *big.Float,
) (*UniswapLiquidity, error) {

	ttb := common.HexToAddress(sn.AddressTargetToken)
	ttbTkn, err := erc20.NewErc20(ttb, e)
	if err != nil {
		return nil, err
	}
	tp := common.HexToAddress(sn.AddressTargetPaired)

	return &UniswapLiquidity{
		ethClient:         e,
		sniperClient:      s,
		sniperTTBAddr:     ttb,
		sniperTTBTkn:      ttbTkn,
		sniperTokenPaired: tp,
		sniperMinLiq:      sn.MinimumLiquidity,
		sniperChainID:     sn.ChainID,
		gasMultiplier:     gasMultiplier,
	}, nil
}

// Add checks when the tx is of addLiquidity in an UniswapV2 AMM fork.
// If all our checks regarding the snipe passes (eg. the devs adding a minimum liquidity that we expect)
// then we invoke the snipe
func (u *UniswapLiquidity) Add(ctx context.Context, tx *types.Transaction, pending bool) error {
	sender, err := u.getTxSenderAddressQuick(tx)
	if err != nil {
		return fmt.Errorf("error getting sender address: %s", err)
	}

	// parse the info of the swap so that we can access it easily
	var addLiquidity = u.newInputFromTx(tx)

	// security checks
	// does the liquidity addition deals with the token i'm targetting?
	if addLiquidity.TokenAddressA == u.sniperTTBAddr || addLiquidity.TokenAddressB == u.sniperTTBAddr {
		// does the liquidity is added on the right pair?
		if addLiquidity.TokenAddressA == u.sniperTokenPaired || addLiquidity.TokenAddressB == u.sniperTokenPaired {
			tknBalanceSender, err := u.sniperTTBTkn.BalanceOf(nil, sender)
			if err != nil {
				return fmt.Errorf("error getting balance of token to buy: %s", err)
			}

			var amountTknMin *big.Int
			var amountPairedMin *big.Int
			if addLiquidity.TokenAddressA == u.sniperTTBAddr {
				amountTknMin = addLiquidity.AmountTokenAMin
				amountPairedMin = addLiquidity.AmountTokenBMin
			} else {
				amountTknMin = addLiquidity.AmountTokenBMin
				amountPairedMin = addLiquidity.AmountTokenAMin
			}
			// we check if the liquidity provider really possess the liquidity he wants to add, because it is possible to be lured by other bots that fake liquidity addition.
			checkBalanceTknLP := amountTknMin.Cmp(tknBalanceSender)
			if checkBalanceTknLP == 0 || checkBalanceTknLP == -1 {
				// we check if the liquidity provider add enough collateral (WBNB or BUSD) as expected by our configuration. Bc sometimes the dev fuck the pleb and add way less liquidity that was advertised on telegram.
				if amountPairedMin.Cmp(u.sniperMinLiq) == 1 {
					log.Info(fmt.Sprintf("snipe executed for tx: %s", tx.Hash().String()))
					return u.sniperClient.Snipe(ctx, u.getGasPrice(tx, pending))
				} else {
					log.Info(fmt.Sprintf(
						"liquidity added but lower than expected: %.4f %s vs %.4f expected",
						formatETHWeiToEther(amountPairedMin),
						u.getTokenSymbol(u.sniperTokenPaired),
						formatETHWeiToEther(u.sniperMinLiq),
					))
				}
			}
		}
	}
	return nil
}

// AddETH checks when the tx is of addLiquidityETH in an UniswapV2 AMM fork.
// If all our checks regarding the snipe passes (eg. the devs adding a minimum liquidity that we expect)
// then we invoke the snipe
// TODO Super similars, refactor?
func (u *UniswapLiquidity) AddETH(ctx context.Context, tx *types.Transaction, pending bool) error {
	// parse the info of the swap so that we can access it easily
	sender, err := u.getTxSenderAddressQuick(tx)
	if err != nil {
		return fmt.Errorf("error getting sender address: %s", err)
	}

	addLiquidity := u.newETHInputFromTx(tx)

	tknBalanceSender, err := u.sniperTTBTkn.BalanceOf(nil, sender)
	if err != nil {
		return fmt.Errorf("error getting balance of token to buy: %s", err)
	}

	checkBalanceLP := addLiquidity.AmountTokenMin.Cmp(tknBalanceSender)

	// security checks:
	// does the liquidity addition deals with the token i'm targetting?
	if addLiquidity.TokenAddress == u.sniperTTBAddr {
		// we check if the liquidity provider really possess the liquidity he wants to add, because it is possible to be lured by other bots that fake liquidity addition.
		if checkBalanceLP == 0 || checkBalanceLP == -1 {
			// we check if the liquidity provider add enough collateral (WBNB or BUSD) as expected by our configuration. Bc sometimes the dev fuck the pleb and add way less liquidity that was advertised on telegram.
			if tx.Value().Cmp(u.sniperMinLiq) == 1 {
				if addLiquidity.AmountETHMin.Cmp(u.sniperMinLiq) == 1 {
					log.Info(fmt.Sprintf("snipe executed for tx: %s", tx.Hash().String()))
					return u.sniperClient.Snipe(ctx, u.getGasPrice(tx, pending))
				}
			} else {
				log.Info(fmt.Sprintf(
					"liquidity added but lower than expected: %.4f vs %.4f expected",
					formatETHWeiToEther(tx.Value()),
					formatETHWeiToEther(u.sniperMinLiq),
				))
			}
		}
	}
	return nil
}

func (u *UniswapLiquidity) newInputFromTx(tx *types.Transaction) uniswapAddLiquidityInput {
	data := tx.Data()[4:]
	tokenA := common.BytesToAddress(data[12:32])
	tokenB := common.BytesToAddress(data[44:64])
	var amountTokenADesired = new(big.Int)
	amountTokenADesired.SetString(common.Bytes2Hex(data[64:96]), 16)
	var amountTokenBDesired = new(big.Int)
	amountTokenBDesired.SetString(common.Bytes2Hex(data[96:128]), 16)
	var amountTokenAMin = new(big.Int)
	amountTokenAMin.SetString(common.Bytes2Hex(data[128:160]), 16)
	var amountTokenBMin = new(big.Int)
	amountTokenBMin.SetString(common.Bytes2Hex(data[160:192]), 16)
	to := common.BytesToAddress(data[204:224])
	var deadline = new(big.Int)
	deadline.SetString(common.Bytes2Hex(data[224:256]), 16)

	return uniswapAddLiquidityInput{
		TokenAddressA:       tokenA,
		TokenAddressB:       tokenB,
		AmountTokenADesired: amountTokenADesired,
		AmountTokenBDesired: amountTokenBDesired,
		AmountTokenAMin:     amountTokenAMin,
		AmountTokenBMin:     amountTokenBMin,
		Deadline:            deadline,
		To:                  to,
	}
}

func (u *UniswapLiquidity) newETHInputFromTx(tx *types.Transaction) uniswapAddLiquidityETHInput {
	data := tx.Data()[4:]
	token := common.BytesToAddress(data[12:32])
	var amountTokenDesired = new(big.Int)
	amountTokenDesired.SetString(common.Bytes2Hex(data[32:64]), 16)
	var amountTokenMin = new(big.Int)
	amountTokenMin.SetString(common.Bytes2Hex(data[64:96]), 16)
	var amountETHMin = new(big.Int)
	amountETHMin.SetString(common.Bytes2Hex(data[96:128]), 16)

	to := common.BytesToAddress(data[140:160])
	var deadline = new(big.Int)
	deadline.SetString(common.Bytes2Hex(data[160:192]), 16)

	return uniswapAddLiquidityETHInput{
		TokenAddress:       token,
		AmountTokenDesired: amountTokenDesired,
		AmountETHMin:       amountETHMin,
		AmountTokenMin:     amountTokenMin,
		Deadline:           deadline,
		To:                 to,
	}
}

func (u *UniswapLiquidity) getTxSenderAddressQuick(tx *types.Transaction) (common.Address, error) {
	msg, err := tx.AsMessage(types.LatestSignerForChainID(u.sniperChainID), nil)
	if err != nil {
		return common.Address{}, err
	}
	return msg.From(), nil
}

func (u *UniswapLiquidity) getTokenSymbol(tokenAddress common.Address) string {
	tokenIntance, _ := erc20.NewErc20(tokenAddress, u.ethClient)
	sym, err := tokenIntance.Symbol(nil)
	if err != nil {
		return fmt.Sprintf("error getting token symbol of %s: %s", tokenAddress.String(), err)
	}
	return sym
}

func (u *UniswapLiquidity) getGasPrice(tx *types.Transaction, pending bool) *big.Int {
	if pending {
		return tx.GasPrice() // frontrunning with same gas price if tx is still in the mempool.
	}

	// If a gas multiplier is provided and wellformed, then increment it.
	// This is useful if the add liquidity tx is already confirmed as we are not frontrunning per-se
	// On a real frontrunning operation this is counter-productive as we would always fail because we frontrun the
	// liquidity addition itself.
	gas := tx.GasPrice()
	if u.gasMultiplier != nil && u.gasMultiplier.Cmp(minGasMultiplier) > 0 {
		ig := new(big.Float).SetInt(gas)
		_, _ = ig.Mul(ig, u.gasMultiplier).Int(gas)
	}
	return gas
}

func formatETHWeiToEther(etherAmount *big.Int) float64 {
	var base, exponent = big.NewInt(10), big.NewInt(18)
	denominator := base.Exp(base, exponent, nil)
	// Convert to float for precision
	tokensSentFloat := new(big.Float).SetInt(etherAmount)
	denominatorFloat := new(big.Float).SetInt(denominator)
	// Divide and return the final result
	final, _ := new(big.Float).Quo(tokensSentFloat, denominatorFloat).Float64()
	return final
}
