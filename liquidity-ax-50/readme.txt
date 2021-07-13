AX-50 Liquidity Sniper

This bot require you to run the GETH client + use the eth-brownie framework. All addresses and private keys contained have been changed for the sake of this public repo.

This is heavily based on https://github.com/Supercycled/cake_sniper, so major thanks to him.

Use at you own risk.

## Global Workflow

AX-50 is a frontrunning bot primarily aimed at liquidity sniping on AMM like PancakeSwap. Liquidity sniping is the most profitable way I found to use it. But you can add pretty much any features involving frontrunning (liquidation, sandwich attacks etc..).

Being able to frontrun implies building on top of GETH client and having access to the mempool. My BSC node was running on AWS, hence the ax-50/global folder that I needed to send back and forth to the server with sniping config.

## Setup

I created the script scheduler.py which will run you through all the necessary steps to configure the bot. The configuration file of the scheduler is variables.py, so please be sure to adapt everything in variables.py to your own configuration.

The scheduler walk you through 4 phases:
- Expectations: helps you calculate the minimal amount of tokens you can expect with the snipe depending on the liquidity addition and your amount of WBNB. Feel free to tweak the variables according to your case and relaunch the script multiples times to test amountOutMin expectations.

- Swarm: helps you by creating the accounts swarm and disperse the BNB/ETH to all of these accounts that will be used by the clogger at the end. You can chose the size of the swarm by a power of 2 depending of the rounds number you parametrize in variables.py. Note that > 256 accounts into the swarm starts too cause instability issues when the bot is triggered. 2 BNB is enough to fund 256 accounts and there is a "refund" function on the sript that allows for all the account to send back their dust BNB/ETH to you whenever you want. There is no such thing as BNB/ETH waste.

- Send the ax-50/global folder to the AWS server (you might not need it)

- Trigger configuration: call configureSnipe on Trigger to arm the bot.

That's it! the bot should be ready to snipe! The bot is currently defined to work with BSC and PancakeSwap. But you can adapt is to whatever EVM blockchain with its equivalent copy of Uniswap V2. To do this, just change the variables in the files variables.py and ax-50/global/config.go
