from brownie import *

MYBUY = 100 # Amount of BNB I plan to snipe
EXTERNAL_BUY = 100 # order size expected from other bots in BNB
RESERVE_IN = 300 # amount of BNB liquidity that will be added on PCS by the team
RESERVE_OUT = 88888 # amount of token liquidity that will be added on PCS by the team

TRIGGER_ADDRESS_MAINNET = "0x39695B38c6d4e5F73acE974Fd0f9F6766c2E5544" # addy changed for public repo
BUSD_WBNB_PAIR_ADDRESS = "0x58F876857a02D6762E0101bb5C46A8c1ED44Dc16"
WBNB_ADDRESS = "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"

# ROUNDS helper:
# 3 --> 8 accounts
# 4 --> 16 accounts
# 5 --> 32 accounts
# 6 --> 64 accounts
# 7 --> 128 accounts
# 8 --> 256 accounts
# 9 --> 512 accounts
BEE_ROUNDS = 4 # swarming : number of dispersion rounds
BEE_BNB_AMOUNT = 1 # swarming : number of BNB to spread
BEEBOOK_PATH  = "./config/bee_book.json"
BEEBOOK_TMP_PATH = "./config/bee_book_temp.json"

# ERC20 addy of the token you want to snipe
TOKEN_TO_BUY_ADDRESS = web3.toChecksumAddress("0x39695B38c6d4e5F73acE974Fd0f9F6766c2E5544")
# How many of wbnb you want to use for the snipe
AMOUNT_IN_WBNB = 150*10**18
# How many tokens you expect from the snipe as a minimal
AMOUNT_OUT_MIN_TKN = 7000*10**18
GWEI = 1000 
PAIRED_TOKEN = WBNB_ADDRESS

# those accounts name are supposed to be registered in your eth-brownie setup files 
DISPERSER_ACCOUNT = "press1"
TRIGGER_OWNER = "bsc2"
TRIGGER_ADMIN = "press1"