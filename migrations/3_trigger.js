const assert = require('assert');

const CustomPCSRouter = artifacts.require("CustomPCSRouter");

const Trigger = artifacts.require("Trigger");

const { addresses } = require('./../config/configurations.json')

module.exports = async function (deployer, network) {
  const router = await CustomPCSRouter.deployed();
  assert(router != null, 'Expected router to be set to a contract');

  await deployer.deploy(Trigger, addresses.wbnb);
  const trigger = await Trigger.deployed();
  
  await trigger.setCustomPCSRouter(router.address)
};
