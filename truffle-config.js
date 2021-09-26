require('dotenv').config();

function hdWalletProviderOptions(privateKeyEnvVarValue, mnemonicPhraseEnvVarValue, otherOpts) {
  const opts = { ...otherOpts };
  if(privateKeyEnvVarValue) {
    opts.privateKeys = [privateKeyEnvVarValue];
  }
  else {
    opts.mnemonic = mnemonicPhraseEnvVarValue;
  }
  return opts;
}

const HDWalletProvider = require('@truffle/hdwallet-provider');

module.exports = {
  /**
   * Networks define how you connect to your ethereum client and let you set the
   * defaults web3 uses to send transactions. If you don't specify one truffle
   * will spin up a development blockchain for you on port 9545 when you
   * run `develop` or `test`. You can ask a truffle command to use a specific
   * network from the command line, e.g
   *
   * $ truffle test --network <network-name>
   */

  networks: {
    // Useful for testing. The `development` name is special - truffle uses it by default
    // if it's defined here and no other network is specified at the command line.
    // You should run a client (like ganache-cli, geth or parity) in a separate terminal
    // tab if you use this network and you must also set the `host`, `port` and `network_id`
    // options below to some value.
    //
    development: {
      host: process.env.ETH_DEV_RPC_HOST || '127.0.0.1',     // Localhost (default: none)
      port: process.env.ETH_DEV_RPC_PORT || 8545,            // Standard Ethereum port (default: none)
      network_id: process.env.ETH_DEV_RPC_NETWORK_ID || '*',       // Any network (default: none)
      gas: parseInt(process.env.ETH_DEV_RPC_GAS, 10) || 999999999 // required for deploy, otherwise it throws weird require-errors on constructor
    },
    bsctestnet: {
      provider: () => new HDWalletProvider(hdWalletProviderOptions(
        process.env.BINANCE_WALLET_PRIVATE_KEY,
        process.env.BINANCE_WALLET_MNEMONIC,
        {
          providerOrUrl: 'https://data-seed-prebsc-2-s2.binance.org:8545/'
        }
      )),
      network_id: 0x61,
      confirmations: 10,
      timeoutBlocks: 200,
      gas: 10000000,
      skipDryRun: true
    },
    bscmainnet: {
      provider: () => new HDWalletProvider(hdWalletProviderOptions(
        process.env.BINANCE_MAINNET_WALLET_PRIVATE_KEY,
        process.env.BINANCE_MAINNET_WALLET_MNEMONIC,
        {
          providerOrUrl: 'https://bsc-dataseed.binance.org/'
        }
      )),
      network_id: 0x38,
      confirmations: 10,
      timeoutBlocks: 200,
      gas: 5600000,
      skipDryRun: true
    },
  },

  // Set default mocha options here, use special reporters etc.
  mocha: {
    // timeout: 100000
  },

  // Configure your compilers
  compilers: {
    solc: {
      version: "0.6.6",    // Fetch exact version from solc-bin (default: truffle's version)
      settings: {          // See the solidity docs for advice about optimization and evmVersion
        optimizer: {
          enabled: true,
          runs: 200
        },
      }
    }
  },
  plugins: [
    "truffle-plugin-verify",
    "truffle-contract-size"
  ],
  api_keys: {
    bscscan: process.env.BSCSCAN_API_KEY
  },
};
