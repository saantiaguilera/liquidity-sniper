import { 
    chain, order, contract, token, accounts
} from '../config/local.json';
import { ethers } from "ethers";
import { BigNumber } from '@ethersproject/bignumber';
import * as readline from 'readline';
import { TransactionRequest } from '@ethersproject/abstract-provider';

const orderSize = order.size;
const minimumTokens = order.expected_tokens;
const { admin } = accounts;

const bscProvider = new ethers.providers.JsonRpcProvider(
    chain.nodes.configure,
    {
        chainId: chain.id,
        name: chain.name,
    }
)

let rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
});

async function applyConfiguration(
    token: ethers.Contract, 
    pair: string,
    orderAmount: BigNumber,
    trigger: ethers.Contract,
    triggerAdminWallet: ethers.Wallet,
    gasPrice: BigNumber,
): Promise<boolean> {

    console.log(`\n> Applying trigger configuration`)
    console.log(`  Using admin: ${triggerAdminWallet.address}`)
    console.log(`  Admin balance: ${((await triggerAdminWallet.getBalance()).div(BigNumber.from(10).pow(14)).toNumber() / 10000).toFixed(3)} BNB`)
    console.log(`  Trigger contract: ${contract.trigger}`)

    // minTokens can have up to 3 decimal places in floating point in case token has low supply
    const minTokens = BigNumber.from(minimumTokens * 1000).mul(BigNumber.from(10).pow(15))

    const { hash } = await trigger.configureSnipe(
        pair,
        orderAmount,
        token.address,
        minTokens,
        {
            from: triggerAdminWallet.address,
            gasPrice: gasPrice,
        }
    )

    console.log(`\n> Trigger configuration submitted: ${hash}`)
    const receipt = await bscProvider.waitForTransaction(hash);
    if (receipt.status != 1) {
        console.log(` [ERROR] Tx ${hash} failed: ${JSON.stringify(receipt)}`)
        return false
    }
    console.log(`  Applied configuration succesfully.`)
    return true
}

async function supplyTrigger(
    orderAmount: BigNumber,
    trigger: ethers.Contract,
    triggerAdminWallet: ethers.Wallet,
    gasPrice: BigNumber,
): Promise<boolean> {

    const wbnbAbi = [
        "function balanceOf(address who) public view returns (uint256)",
    ]
    const wbnbAddress = token.wbnb
    const wbnb = new ethers.Contract(wbnbAddress, wbnbAbi, bscProvider)
    const triggerBalance = await wbnb.balanceOf(contract.trigger)

    if (triggerBalance.lt(orderAmount)) {
        console.log(`\n> Supplying BNB to trigger contract`)
        const diffAmount = orderAmount.sub(triggerBalance).add(1)
        if ((await triggerAdminWallet.getBalance()).lte(diffAmount.add(BigNumber.from(21000).mul(gasPrice)))) {
            console.log(`  [ERROR] Trigger admin ${triggerAdminWallet.address} has insufficient balance to provide to sniper. Required: ${((diffAmount.add(BigNumber.from(21000).mul(gasPrice)).div(BigNumber.from(10).pow(14)).toNumber()) / 10000).toFixed(3)} BNB`)
            return false
        }

        const txReq: TransactionRequest = {
            to: contract.trigger,
            value: diffAmount,
            gasPrice: gasPrice,
            gasLimit: ethers.utils.hexlify(300000),
        }

        const { hash } = await triggerAdminWallet.sendTransaction(txReq)
    
        console.log(`  Tx supplying BNB for trigger contract ${contract.trigger}: ${hash}`)
        const receipt = await bscProvider.waitForTransaction(hash);
        if (receipt.status != 1) {
            console.log(` [ERROR] Tx ${hash} failed at ${triggerAdminWallet.address}: ${JSON.stringify(receipt)}`)
            return false
        }
        console.log(`  Trigger supplied with necessary BNB.`)
    }
    return true
}

async function configureTrigger(token: ethers.Contract, pair: string): Promise<void> {
    const triggerAdminWallet = new ethers.Wallet(admin, bscProvider)
    const triggerAbi = [
        "function configureSnipe(address _tokenPaired, uint _amountIn, address _tknToBuy, uint _amountOutMin) external returns(bool)",
    ]
    const trigger = new ethers.Contract(contract.trigger, triggerAbi, triggerAdminWallet)
    const orderAmount = BigNumber.from(orderSize * 1000).mul(BigNumber.from(10).pow(15)) // orderSize can have up to 3 decimal places
    const gasPrice = await bscProvider.getGasPrice()

    let ok = await applyConfiguration(
        token,
        pair,
        orderAmount,
        trigger,
        triggerAdminWallet,
        gasPrice,
    )
    if (!ok) {
        console.log('[ERROR] Halting.')
        return
    }
    
    ok = await supplyTrigger(orderAmount, trigger, triggerAdminWallet, gasPrice)
    if (!ok) {
        console.log('[ERROR] Halting.')
        return
    }
}

async function promptTrigger(): Promise<void> {
    const erc20Abi = [
        "function symbol() view returns (string)",
    ]
    const erc20 = new ethers.Contract(token.address, erc20Abi, bscProvider)
    const tokenSymbol = await erc20.symbol()

    console.log('> Preparing to configure trigger')
    console.log(`  Token to buy: ${erc20.address}`)
    console.log(`  Order size: ${orderSize} BNB`)
    console.log(`  Min buy: ${minimumTokens} ${tokenSymbol}`)
    console.log('[WARNING] Configuring a trigger will REMOVE any existing ones. Make sure the previous trigger has been already used.')

    rl.question(`\n> Configure new trigger? [y/n]: `, async (answer) => {
        switch(answer.toLowerCase()) {
          case 'y':
            await configureTrigger(erc20, token.pair_address)
            break;
          default:
            console.log('  Configure trigger process ends now.');
        }
        rl.close();
    });
}

promptTrigger()