from brownie import *
from math import floor
from time import sleep
import itertools
from variables import *
import itertools
import json
import os
import concurrent.futures
from pynput.keyboard import Key, Controller

COUNTER = itertools.count()
ACC_LIST = []
ACC_INDEX = itertools.count()

# Expectations

def _bnb_price():
    assert chain.id == 56, "_bnbPrice: WRONG NETWORK. This function only works on bsc mainnet"
    pair_busd = interface.IPancakePair(busd_wbnb_addr)
    (reserveUSD, reserveBNB, _) = pair_busd.getReserves()
    price_busd = reserveBNB / reserveUSD
    return round(price_busd, 2)

def _quote(amin, reserve_in, reserve_out):
    if reserve_in == 0 and reserve_out == 0:
        return 'empty reserves, no quotation'
    amount_in_with_fee = amin * 998
    num = amount_in_with_fee * reserve_out
    den = reserve_in * 1000 + amount_in_with_fee
    return round(num / den)

def _expectations(my_buy, external_buy, reserve_in, reserve_out, queue_number):
    i = 1
    add_in = 0
    sub_out = 0
    while i < queue_number:
        amout = _quote(external_buy, reserve_in + add_in, reserve_out - sub_out)
        add_in += external_buy
        sub_out += amout
        i += 1
    bought_tokens = _quote(my_buy, reserve_in + add_in, reserve_out - sub_out)
    price_per_token = my_buy / bought_tokens
    return bought_tokens, price_per_token, add_in

def expectations(my_buy, external_buy, reserve_in, reserve_out, base_asset = "BNB"):

    bnb_p = _bnb_price()
    print(
        f'--> if the liq added is {reserve_in} BNB / {reserve_out} tokens and I want to buy with {my_buy} BNB : \n')
    for i in range(1, 30, 1):
        (bought_tokens, price_per_token, add_in) = _expectations(
            my_buy, external_buy, reserve_in, reserve_out, i)

        if base_asset == "BNB":
            print(
                f'amount bought: {bought_tokens} | {round(price_per_token, 5)} BNB/tkn | {round(price_per_token * bnb_p, 7) } $/tkn | , capital entered before me: {add_in} BNB')
        else:
            print(
                f'amount bought: {bought_tokens} | {round(price_per_token, 5)} BNB/tkn| , capital entered before me: {add_in} BNB')
    print(f'\n--> BNB price: {bnb_p} $')
    print("WARNING: exit and restart brownie to be sure variables corrections are taken into account!\n")

    input("Press any key to continue, or ctrl+c to stop and try other expectation parameters")

# Swarmer

def create_temp_address_book(tmp_path):
    """create the temporary csv file that store addresses"""
    try:
        os.remove(tmp_path)
    except:
        pass
    finally:
        with open(tmp_path, "w"):
            pass


def save_address_book(tmp_path, path):
    print("---> Saving address book...")
    with open(tmp_path, "r") as address_book:
        data = json.load(address_book)
        for account in data:
            addr = account["address"]
            balance = accounts.at(addr).balance() / 10**18
            account["balance"] = balance

    with open(path, "w") as final_address_book:
        json.dump(data, final_address_book, indent=2)
    print("Done!")

def create_account():
    idx = next(ACC_INDEX)
    new_account = web3.eth.account.create()
    new_account = accounts.add(new_account.key.hex())
    pk = new_account.private_key
    account_dict = {
        "id": idx,
        "address": new_account.address,
        "pk": pk
    }
    ACC_LIST.append(account_dict)
    return new_account


def swarming(acc):
    sleep(10)
    new_account = create_account()
    pk = acc["pk"]
    bee = accounts.add(pk)
    tx = bee.transfer(
        to=new_account.address,
        amount=bee.balance() // 2,
        silent=True,
        gas_limit=22000,
        allow_revert=True)
    return f'bee{acc["id"]} --> paid {tx.value / 10**18} BNB to new_account'


def _initSwarm(tmp_path, path, rounds, bnb_amount):
    create_temp_address_book(tmp_path)
    print("(admin account)")
    me = accounts.load(DISPERSER_ACCOUNT)
    old_balance = me.balance()
    print(f'\n--> seed account balance: {old_balance/10**18} BNB\n')

    account0 = create_account().address
    print("\nCREATING ACCOUNTS SWARM...\n")
    tx = me.transfer(to=account0, amount=f'{bnb_amount} ether', silent=True)
    print(f'seed --> paid {tx.value / 10**18} BNB to new_account')

    # spreading bnb among the swarm
    COUNTER = itertools.count()
    for _ in range(rounds):
        n = next(COUNTER)
        print(f'\nROUND n°{n}\n')
        tmp_acc_list = ACC_LIST.copy()

        with concurrent.futures.ThreadPoolExecutor() as executor:
            results = [executor.submit(swarming, acc)
                       for acc in tmp_acc_list]
            for f in concurrent.futures.as_completed(results):
                print(f.result())

    with open(tmp_path, "a") as address_book:
        json.dump(ACC_LIST, address_book, indent=2)

    print('\nSWARM CREATED!\n')
    print(f'Total accounts created: {len(ACC_LIST)}\n')
    save_address_book(tmp_path, path)

def _refund(entry, me):
    pk = entry["pk"]
    acc = accounts.add(pk)
    if acc.balance() > 0:
        tx = acc.transfer(me, amount=acc.balance() -
                          21000 * 10**10, required_confs=0, silent=True)
        return f'bee{entry["id"]} --> paid {tx.value/10**18} to seed address'
    else:
        return "empty balance"


def refund(path):
    me = accounts.load('press1')

    with open(path, "r") as book:
        data = json.load(book)

        with concurrent.futures.ThreadPoolExecutor() as executor:

            results = [executor.submit(_refund, acc, me)
                       for acc in data]
            for f in concurrent.futures.as_completed(results):
                print(f.result())

    pending = [True]
    while True in pending:
        pending.clear()
        for tx in history:
            pending.append(tx.status == -1)
        print(f'remaining pending tx: {pending.count(True)}')
        sleep(1)

    print(f'\nREFUND DONE! --> seed balance : {me.balance()/10**18} BNB')


def _checkBalances(entry):
    pk = entry["pk"]
    acc = accounts.add(pk)
    balance = acc.balance()
    if balance / 10**18 > 0.0002:

        print(f'bee{entry["id"]} : non empty balance: {balance/10**18} BNB')
        return balance, 1
    else:
        return 0, 0


def swarmer(tmp_path, path, rounds, bnb_amount):
    print("Checking for existing, non empty address book...")
    with open(path, "r") as book:
        data = json.load(book)

        total_dust = 0
        total_nonempty_bee = 0

        for entry in data:
            (balance, bee) = _checkBalances(entry)
            total_dust += balance
            total_nonempty_bee += bee

    print(
        f'\nFound an already existing address book with {total_nonempty_bee} non empty balance addresses')
    print(f'Total BNB to claim: {total_dust/10**18}\n')

    if total_dust > 0:
        ipt = input("Launch refund? ('y' for yes, any other key for no)")
        if ipt.lower() == "y":
            refund(path)
        else:
            return

    print(
        f'\nReady to launch new swarm. Parameters:\n\t- Rounds: {rounds} ({2**rounds} addresses)\n\t- Number of BNB to spread: {bnb_amount}\n')
    ipt = input("Initialise new swarm? ('y' for yes, any other key for no)")

    if ipt.lower() == "y":
        _initSwarm(tmp_path, path,rounds, bnb_amount )
    else:
        return


def createBeeBook():
    swarmer(BEEBOOK_TMP_PATH, BEEBOOK_PATH, BEE_ROUNDS, BEE_BNB_AMOUNT )

# Trigger

def configureTrigger():
    tokenToBuy = interface.ERC20(ttb_addr)

    print(
        f'\nCURRENT CONFIGURATION:\n\nWANT TO BUY AT LEAST {AMOUNT_OUT_MIN_TKN/10**18} {tokenToBuy.name()} (${tokenToBuy.symbol()})\nWITH {AMOUNT_IN_WBNB / 10**18} WBNB\n')
    ipt = input(
        "---> If this is ok, press 'y' to call configureSnipe, any other key to skip")

    if ipt.lower() == 'y':

        print("\n---> loading TRIGGER owner and admin wallet:")
        print("(owner pwd)")
        me = accounts.load(TRIGGER_OWNER)
        print("(admin pwd)")
        admin = accounts.load(TRIGGER_ADMIN)
        tkn_balance_old = tokenToBuy.balanceOf(admin)

        print("\n---> configuring TRIGGER for sniping")
        trigger = interface.ITrigger2(trigger_addr)
        trigger.configureSnipe(PAIRED_TOKEN, AMOUNT_IN_WBNB,
                               ttb_addr, AMOUNT_OUT_MIN_TKN, {'from': me, "gas_price": "10 gwei"})
        

        triggerBalance = interface.ERC20(wbnb_addr).balanceOf(trigger)

        if triggerBalance < AMOUNT_IN_WBNB:

            amountToSendToTrigger = AMOUNT_IN_WBNB - triggerBalance + 1
            assert me.balance() >= amountToSendToTrigger + 10**18 , "STOPING EXECUTION: TRIGGER DOESNT HAVE THE REQUIRED WBNB AND OWNER BNB BALANCE INSUFFICIENT!"

            print(f'---> transfering {amountToSendToTrigger / 10**18} BNB to TRIGGER')
            
            me.transfer(trigger, amountToSendToTrigger)

        config = trigger.getSnipeConfiguration({'from': me})
        assert config[0] == PAIRED_TOKEN
        assert config[1] == AMOUNT_IN_WBNB
        assert config[2] == ttb_addr
        assert config[3] == AMOUNT_OUT_MIN_TKN

        print("\nTRIGGER CONFIGURATION READY\n")
        print(
            f'---> Wbnb balance of trigger: {interface.ERC20(wbnb_addr).balanceOf(trigger)/10**18}')
        print(
            f'---> Token balance of admin: {tkn_balance_old/10**18 if tkn_balance_old != 0 else 0}\n\n')

def main():
    print("\n///////////// EXPECTATION PHASE //////////////////////////\n")
    expectations(mbuy, ext_buy, reserve_in, reserve_out)
    print("\n///////////// BEE BOOK CREATION PHASE //////////////////////////////\n")
    createBeeBook()
    print("\n///////////// TRIGGER CONFIGURATION PHASE /////////////////////\n")
    configureTrigger()
