import { 
    chain, contract, token, accounts
} from '../config/local.json';
import { ethers } from "ethers";
import { BigNumber } from '@ethersproject/bignumber';
import * as readline from 'readline';
import { exit } from 'process';

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

async function withdrawTrigger(tokenAddress: string, amount: BigNumber): Promise<void> {
    const triggerAdminWallet = new ethers.Wallet(admin, bscProvider)
    const triggerAbi = [
        "function emmergencyWithdrawTkn(address _token, uint _amount) external returns(bool)",
    ]
    const trigger = new ethers.Contract(contract.trigger, triggerAbi, triggerAdminWallet)
    const gasPrice = await bscProvider.getGasPrice()

    const { hash } = await trigger.emmergencyWithdrawTkn(
        tokenAddress,
        amount,
        {
            from: triggerAdminWallet.address,
            gasPrice: gasPrice,
        }
    )

    console.log(`Trigger withdrawal submitted: ${hash}`)
    const receipt = await bscProvider.waitForTransaction(hash);
    if (receipt.status != 1) {
        console.log(` [ERROR] Tx ${hash} failed: ${JSON.stringify(receipt)}`)
        return
    }
    console.log(`  Withdrawal successful.`)
}

async function promptWithdrawal(): Promise<void> {
    const wbnbAbi = [
        "function balanceOf(address who) public view returns (uint256)",
    ]
    const wbnbAddress = token.wbnb
    const wbnb = new ethers.Contract(wbnbAddress, wbnbAbi, bscProvider)

    console.log('> Checking trigger balance')

    const triggerBalance: BigNumber = await wbnb.balanceOf(contract.trigger)
    console.log(`  WBNB: ${(triggerBalance.div(BigNumber.from(10).pow(14)).toNumber() / 10000).toFixed(3)}`)

    if (triggerBalance.gt(BigNumber.from(10).pow(15))) { // > 0.001
        rl.question(`\n> Withdraw all WBNB? [y/n]: `, async (answer) => {
            switch(answer.toLowerCase()) {
              case 'y':
                await withdrawTrigger(wbnb.address, triggerBalance)
                break;
              default:
                console.log('  Withdrawal process ends now.');
            }
            rl.close();
        });
    } else {
        console.log("No balance to withdraw.")
        exit(0)
    }
}

promptWithdrawal()