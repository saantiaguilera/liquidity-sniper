const assert = require('assert');

const CustomPCSRouter = artifacts.require("CustomPCSRouter");

const Trigger = artifacts.require("Trigger");

module.exports = async function (deployer, network) {
  const router = await CustomPCSRouter.deployed();
  assert(router != null, 'Expected router to be set to a contract');

  await deployer.deploy(Trigger);
  const trigger = await Trigger.deployed();
  
  await trigger.setCustomPCSRouter(router.address)
};
