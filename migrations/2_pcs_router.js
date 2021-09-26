const CustomPCSRouter = artifacts.require("CustomPCSRouter");

module.exports = async function (deployer, network, accounts) {
  await deployer.deploy(CustomPCSRouter);
  
  await CustomPCSRouter.deployed();
};