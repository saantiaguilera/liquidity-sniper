const CustomPCSRouter = artifacts.require("CustomPCSRouter");

const { addresses } = require('./../config/configurations.json')

module.exports = async function (deployer, network, accounts) {
  await deployer.deploy(CustomPCSRouter, addresses.cake_factory, addresses.wbnb);
  
  await CustomPCSRouter.deployed();
};