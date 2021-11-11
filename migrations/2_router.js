const CustomRouter = artifacts.require("CustomRouter");

const { contract, token } = require('../config/local.json')

module.exports = async function (deployer, network, accounts) {
  await deployer.deploy(CustomRouter, contract.factory, token.wbnb, contract.factory_creation_code);
  
  await CustomRouter.deployed();
};