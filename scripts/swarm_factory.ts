import { 
    chain, swarm, accounts
} from '../config/local.json';
import * as fs from 'fs';
import { ethers } from "ethers";
import { BigNumber } from '@ethersproject/bignumber';
import * as readline from 'readline';
import { TransactionRequest } from '@ethersproject/abstract-provider';

const { rounds, path, spread_amount } = swarm;
const { disperser } = accounts;

const bscProvider = new ethers.providers.JsonRpcProvider(
    chain.node, 
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

function newBook(wallets: Array<ethers.Wallet>) {
    if (fs.existsSync(path)) {
        fs.rmSync(path)
    }

    const stream = fs.createWriteStream(path)
    const res: Array<Bee> = wallets.map(wallet => {
        return {
            pk: wallet.privateKey,
            addr: wallet.address
        }
    })

    stream.write(JSON.stringify(res, null, 2))
    stream.end()
}

function newAccount(): ethers.Wallet {
    return ethers.Wallet.createRandom()
}

async function disperse(wallet: ethers.Wallet, amount: BigNumber): Promise<ethers.Wallet | null> {
    const bee = newAccount()

    const txReq: TransactionRequest = {
        to: bee.address,
        value: amount.sub(21000 * 6**10),
        gasPrice: ethers.utils.parseUnits("6", "gwei"),
        gasLimit: ethers.utils.hexlify(21000),
    }
    const { hash } = await wallet.sendTransaction(txReq)

    console.log(`  New bee created: ${bee.address}`)
    console.log(`  Tx supplying BNB for bee ${bee.address}: ${hash}`)
    const receipt = await bscProvider.waitForTransaction(hash);
    if (receipt.status != 1) {
        console.log(` [WARNING] Tx ${hash} failed at ${wallet.address} for new bee ${bee.address}: ${receipt}`)
        return null
    }
    return bee
}

async function createSwarm(): Promise<void> {
    let wallets: Array<ethers.Wallet> = []

    const disperserWallet = new ethers.Wallet(disperser, bscProvider)
    console.log('> Creating swarm...')
    console.log(`  Disperser wallet: ${disperserWallet.address}`)
    console.log(`  Disperser wallet balance: ${(await disperserWallet.getBalance()).div(10**18).toString()} BNB`)

    // Initial dispersion. Give to the first bee all our snipe amount
    const initialBee = await disperse(disperserWallet, BigNumber.from(spread_amount).mul(10**18))
    if (initialBee != null) {
        wallets.push(initialBee)
    }

    // Round start. On each round pow2 wallets where each one will give half of its amount to the new one
    for (let i = 0; i < rounds; i++) {
        let dispersions: Array<Promise<ethers.Wallet | null>> = []
        wallets.forEach(async wallet => {
            const balance = await wallet.getBalance()
            dispersions.push(disperse(wallet, balance.div(2)))
        })

        const newWallets = await Promise.all(dispersions)
        wallets.push(...newWallets.flatMap(v => !!v ? [v] : []))
    }

    // Save new book
    if (wallets.length > 0) {
        newBook(wallets)
    }
}

async function checkSwarm(): Promise<void> {
    console.log('> Ready to launch new swarm')
    console.log(`  Rounds: ${rounds}`)
    console.log(`  Addresses: ${2**rounds}`)
    console.log(`  BNB to spread: ${spread_amount}`)
    console.log('[WARNING] Creating a new swarm will REMOVE any existing one. Make sure to refund previous ones before creating a new one')

    rl.question(`> Create new swarm? [y/n]`, async (answer) => {
        switch(answer.toLowerCase()) {
          case 'y':
            await createSwarm()
            break;
          default:
            console.log('  Swarm process ends now.');
        }
        rl.close();
    });
}

checkSwarm()