const assert = require('assert');

const CustomRouter = artifacts.require("CustomRouter");

const Trigger = artifacts.require("Trigger");

const { token } = require('./../config/local.json')

module.exports = async function (deployer, network) {
  const router = await CustomRouter.deployed();
  assert(router != null, 'Expected router to be set to a contract');

  await deployer.deploy(Trigger, token.wbnb);
  const trigger = await Trigger.deployed();
  
  await trigger.setCustomRouter(router.address)
};
