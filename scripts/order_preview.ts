import { chain, order, token, previewer } from '../config/local.json';
import { ethers } from "ethers";

const { ext_order_size, liquidity_in_bnb, liquidity_in_token } = previewer;
const tokenAddress = token.address
const selfOrderSize = order.size

const bscProvider = new ethers.providers.JsonRpcProvider(
    chain.node, 
    {
        chainId: chain.id,
        name: chain.name,
    }
)

async function getBNBPrice(): Promise<number> {
    const abi = [
        "function getReserves() external view returns (uint112, uint112, uint32)",
    ]
    const addr = '0x58F876857a02D6762E0101bb5C46A8c1ED44Dc16'
    const busdBNBPair = new ethers.Contract(addr, abi, bscProvider);

    const [ rsvBusd, rsvBNB ] = await busdBNBPair.getReserves()

    const precision = 10000
    return rsvBNB.div(rsvBusd).mul(precision).toNumber() / precision
}

function quote(minAmount: number, rsvIn: number, rsvOut: number): number {
    if (rsvIn == 0 && rsvOut == 0) {
        return 0 // empty reserves, no quotation
    }

    const amountWithFee = minAmount * 998
    return Math.round((amountWithFee * rsvOut) / ((rsvIn * 1000) + amountWithFee))
}

function runSimulation(queueNumber: number): [number, number, number] {
    let i = 1
    let addIn = 0
    let subOut = 0

    while (i < queueNumber) {
        const amount = quote(ext_order_size, liquidity_in_bnb + addIn, liquidity_in_token - subOut)
        addIn += ext_order_size
        subOut += amount
        i++
    }

    const boughtTokens = quote(selfOrderSize, liquidity_in_bnb + addIn, liquidity_in_token - subOut)
    const tokenPrice = selfOrderSize / boughtTokens
    return [ boughtTokens, tokenPrice, addIn ]
}

async function preview(): Promise<void> {
    const erc20Abi = [
        "function symbol() view returns (string)"
    ]
    const erc20 = new ethers.Contract(tokenAddress, erc20Abi, bscProvider)
    const tokenName = await erc20.symbol()

    console.log(`> Expecting liquidity of ${liquidity_in_bnb} BNB + ${liquidity_in_token} ${tokenName}.`)
    console.log(`> [PREVIEW] Order size to snipe: ${selfOrderSize} BNB`)
    
    const bnbPrice = await getBNBPrice()
    console.log(`Current BNB price: ${bnbPrice.toFixed(3)}`)
    console.log('\nStarting simulation...')

    for (let i = 1; i < 30; i++) {
        const [ boughtTokens, tokenPrice, addIn ] = runSimulation(i)

        // capital entered before we get to snipe. Meaning: tx mempool -> [ liqAdd, someBotSnipes, someBotSnipes, weSnipe ]
        console.log(`\n# When ${addIn} BNB are frontrunned before ourselves`)
        console.log(`    Amount bought: ${boughtTokens} ${tokenName}`)
        console.log(`    Token value: ${(tokenPrice/bnbPrice).toFixed(8)} usd/${tokenName}`)
        console.log(`    Pair price: ${tokenPrice.toFixed(5)} BNB/${tokenName}`)
    }
}

preview()