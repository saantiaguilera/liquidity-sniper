package config

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	erc202 "github.com/saantiaguilera/liquidity-ax-50/third_party/erc20"
	uniswap2 "github.com/saantiaguilera/liquidity-ax-50/third_party/uniswap"
	"io/ioutil"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Dark forester account is the account that owns the Trigger and SandwichRouter contract and can configure it beforehand.
// It is also the dest account for the sniped tokens.

var accountAddress = "0x81F37cc0EcAE1dD1c89D79A98f857563873cFA76"
var accountPk = "de8c0753508570d6bc3aea027a5896401c82fe997d3717d19c785Fbbee128695"
var AX_50_ACCOUNT Account

///////// CONST //////////////////
var WBNB_ADDRESS = common.HexToAddress("0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c")
var BUSD_ADDRESS = common.HexToAddress("0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56")
var CAKE_FACTORY_ADDRESS = common.HexToAddress("0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73")
var CAKE_ROUTER_ADDRESS = "0x10ED43C718714eb63d5aA57B78B54704E256024E"
var WBNBERC20 *erc202.Erc20
var FACTORY *uniswap2.IPancakeFactory
var CHAINID = big.NewInt(56)
var STANDARD_GAS_PRICE = big.NewInt(5000000000) // 5 GWEI
var Nullhash = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")

///////// OTHER CONFIGS //////////

// Allows monitoring of tx comming from and to a list of monitored addresses defined in config/address_list.json
var ADDRESS_MONITOR bool = false

// Allows monitoring of any big BNB transfers on BSC
var BIG_BNB_TRANSFER bool = false

///////// SNIPE CONFIG //////////

// activate or not the liquidity sniping
var Sniping bool = true
var PCS_ADDLIQ bool = Sniping

// address of the Trigger smart contract
var TRIGGER_ADDRESS = common.HexToAddress("0xaE23a2ADb82BcF36A14D9c158dDb1E0926263aFC")

// you can choose the base currency. 99% it's WBNB but sometimes it's BUSD
var TOKENPAIRED = WBNB_ADDRESS // base currency
var Snipe SnipeConfiguration

// targeted token to buy (BEP20 address)
var TTB = "0x9412F9AB702AfBd805DECe8e0627427461eF0602"

// ML= minimum liquidity expected when dev add liquidity. We don't want to snipe if the team doesn't add the min amount of liq we expect. it's an important question to solve in the telegram of the project. You can also mmonitor bscscan and see the repartition of WBNB among the address that hold the targeted token and deduce the WBNB liq that wiill be added.
var ML = 200

///////// BIG TRANSFERS CONFIG //////////
var BNB = "50000000000000000000" // 50 BNB
var BigTransfer big.Int
var AddressesWatched = make(map[common.Address]AddressData)

///////////// TYPES /////////////////
type SnipeConfiguration struct {
	TokenAddress common.Address // token address to monitor
	TokenPaired  common.Address
	Tkn          *erc202.Erc20
	MinLiq       *big.Int // min liquidity that will be added to the pool
}

type Address struct {
	Name string
	Addr string
}

type AddressData struct {
	Name    string
	Watched bool
}

///////////// INITIIALISER FUNCS /////////////////
func _initConst(client *ethclient.Client) {
	AX_50_ACCOUNT.Address = common.HexToAddress(accountAddress)
	AX_50_ACCOUNT.Pk = accountPk
	rawPk, err := crypto.HexToECDSA(accountPk)
	if err != nil {
		log.Printf("error decrypting AX_50_ACCOUNT_ACCOUNT pk: %v", err)
	}
	AX_50_ACCOUNT.RawPk = rawPk

	factory, err := uniswap2.NewIPancakeFactory(CAKE_FACTORY_ADDRESS, client)
	if err != nil {
		log.Fatalln("InitFilters: couldn't embed FACTORY: ", err)
	}
	FACTORY = factory

	wbnb, err := erc202.NewErc20(WBNB_ADDRESS, client)
	if err != nil {
		log.Fatalln("InitFilters: couldn't fetch WBNB token: ", err)
	}
	WBNBERC20 = wbnb

	busd, err := erc202.NewErc20(BUSD_ADDRESS, client)
	if err != nil {
		log.Fatalln("InitFilters: couldn't fetch BUSD token: ", err)
	}
}

func _initSniper(client *ethclient.Client) {
	if Sniping == true {
		Snipe.TokenAddress = common.HexToAddress(TTB)
		Snipe.TokenPaired = TOKENPAIRED

		mul10pow14, _ := new(big.Int).SetString("100000000000000", 10)
		MLx10000 := 10000 * ML
		ml := big.NewInt(int64(MLx10000))
		ml.Mul(ml, mul10pow14)
		Snipe.MinLiq = ml
		fmt.Println(Snipe.MinLiq)

		tkn, err := erc202.NewErc20(common.HexToAddress(TTB), client)
		if err != nil {
			log.Fatalln("InitFilters: couldn't fetch token: ", err)
		}
		Snipe.Tkn = tkn
	}
}

func InitDF(client *ethclient.Client) {
	_initConst(client)
	_initSniper(client)

	// initialize BIG_BNB_TRANSFER
	if BIG_BNB_TRANSFER == true {
		bnb, _ := new(big.Int).SetString(BNB, 10)
		BigTransfer = *bnb
	}

	// INITIALISE ADDRESS_MONITOR
	if ADDRESS_MONITOR == true {
		var AddressList []Address
		data, err := ioutil.ReadFile("./config/address_list.json")
		if err != nil {
			log.Fatalln("cannot load address_list.json", err)
		}
		err = json.Unmarshal(data, &AddressList)
		if err != nil {
			log.Fatalln("cannot unmarshall data into AddressList", err)
		}
		for _, a := range AddressList {
			ad := AddressData{a.Name, true}
			AddressesWatched[common.HexToAddress(a.Addr)] = ad
		}
	}
}

// Look onchain for the Trigger contract and return its WBNB balance
func GetTriggerWBNBBalance() *big.Int {
	balance, err := WBNBERC20.BalanceOf(&bind.CallOpts{}, TRIGGER_ADDRESS)
	if err != nil {
		log.Fatalln("couldn't fetch wbnb balance of trigger: ", err)
	}
	return balance
}
