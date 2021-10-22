const CustomRouter = artifacts.require("CustomRouter");

const { contract, token } = require('../config/local.json')

module.exports = async function (deployer, network, accounts) {
  await deployer.deploy(CustomRouter, contract.factory, token.wbnb);
  
  await CustomRouter.deployed();
};