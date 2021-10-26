import { 
    chain, swarm, accounts
} from '../config/local.json';
import * as fs from 'fs';
import { ethers } from "ethers";
import { BigNumber } from '@ethersproject/bignumber';
import * as readline from 'readline';
import { TransactionRequest } from '@ethersproject/abstract-provider';
import { exit } from 'process';

const { path } = swarm;
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

interface Bee {
    readonly pk: string
    readonly addr: string
}

async function refund(me: ethers.Wallet, bee: Bee): Promise<any> {
    const beeWallet = new ethers.Wallet(bee.pk, bscProvider)

    const bal = await beeWallet.getBalance()
    if (bal.gt(0)) {
        const gasPrice = await bscProvider.getGasPrice()
        const txReq: TransactionRequest = {
            to: me.address,
            value: bal.sub(BigNumber.from(21000).mul(gasPrice)),
            gasPrice: gasPrice,
            gasLimit: ethers.utils.hexlify(21000),
        }
        const { hash } = await beeWallet.sendTransaction(txReq)

        console.log(`  Tx for bee ${bee.addr}: ${hash}`)
        const receipt = await bscProvider.waitForTransaction(hash);
        if (receipt.status != 1) {
            console.log(` [WARNING] Tx ${hash} failed at ${bee.addr}: ${JSON.stringify(receipt)}`)
        }
    }
}

async function refundAll(book: Array<Bee>): Promise<void> {
    console.log("\n> Starting refund process..")

    const me = new ethers.Wallet(admin, bscProvider)
    console.log(`  Owner wallet: ${me.address}`)
    console.log(`  Owner wallet balance: ${((await me.getBalance()).div(BigNumber.from(10).pow(14)).toNumber() / 10000).toFixed(3)} BNB`)
    let refunds: Array<Promise<any>> = []

    for (const bee of book) {
        refunds.push(refund(me, bee))
    }

    await Promise.all(refunds)

    console.log(`  New owner wallet balance: ${((await me.getBalance()).div(BigNumber.from(10).pow(14)).toNumber() / 10000).toFixed(3)} BNB`)
    console.log("\n> Refund finished.")
}

async function checkBalance(bee: Bee): Promise<BigNumber> {
    const acc = new ethers.Wallet(bee.pk, bscProvider)
    const balance = await acc.getBalance()
    const minDust = BigNumber.from(2).mul(BigNumber.from(10).pow(14))
    if (balance.gt(minDust)) {
        return balance
    }
    return BigNumber.from(0) // if dust <= 0.0002 consider it a waste
}

async function checkRefund(): Promise<void> {
    if (!fs.existsSync(path)) {
        console.log(`> Bee book doesn't exist`)
        return
    }

    let dust = BigNumber.from(0)
    let fundedBees = 0
    const book: Array<Bee> = JSON.parse(fs.readFileSync(path).toString())

    console.log('> Looking in bee book for BNB dust')
    for (const beeEntry of book) {
        const balance = await checkBalance(beeEntry)

        dust = dust.add(balance)
        if (balance.gt(0)) {
            console.log(`  Account ${beeEntry.addr} holds dust: ${(balance.div(BigNumber.from(10).pow(14)).toNumber() / 10000).toFixed(3)} BNB`)
            fundedBees++
        }
    }

    if (dust.gt(0)) {
        rl.question(`\n> Found ${fundedBees} wallets with BNB dust. Launch refund? [y/n]: `, async (answer) => {
            switch(answer.toLowerCase()) {
              case 'y':
                await refundAll(book)
                break;
              default:
                console.log('  Refund canceled.');
            }
            rl.close();
        });
    } else {
        console.log("No dust found.")
        exit(0)
    }
}

checkRefund()