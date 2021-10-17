// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.6.0 <0.8.0;

import "@openzeppelin/contracts/utils/Context.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

interface IWBNB {
    function withdraw(uint) external;
    function deposit() external payable;
}

interface ICustomPCSRouter {
    function swapExactTokensForTokens(
        uint amountIn,
        uint amountOutMin,
        address[] calldata path,
        address to,
        uint deadline
    ) external returns (uint[] memory amounts);
}

contract Trigger is Ownable {

    // bsc variables 
    address constant wbnb = 0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c;
    address constant cakeFactory = 0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73;

    // eth variables 
    // address constant wbnb= 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;
    // address constant cakeRouter = 0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D;
    // address constant cakeFactory = 0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f;
    
    address payable private administrator;
    address private customRouter;

    uint private wbnbIn;
    uint private minTknOut;

    address private tokenToBuy;
    address private tokenPaired;

    bool private snipeLock;

    constructor() public {
        administrator = payable(msg.sender);
    }
    
    receive() external payable {
        IWBNB(wbnb).deposit{value: msg.value}();
    }

    // Trigger is the smart contract in charge or performing liquidity sniping.
    // Its role is to hold the BNB, perform the swap once ax-50 detect the tx in the mempool and if all checks are passed; then route the tokens sniped to the owner.
    // It requires a first call to configureSnipe in order to be armed. Then, it can snipe on whatever pair no matter the paired token (BUSD / WBNB etc..).
    // This contract uses a custtom router which is a copy of PCS router but with modified selectors, so that our tx are more difficult to listen than those directly going through PCS router.
    
    // perform the liquidity sniping
    function snipeListing() external returns(bool success) {
        require(IERC20(wbnb).balanceOf(address(this)) >= wbnbIn, "snipe: not enough wbnb on the contract");
        IERC20(wbnb).approve(customRouter, wbnbIn);
        require(snipeLock == false, "snipe: sniping is locked. See configure");
        snipeLock = true;
        
        address[] memory path;
        if (tokenPaired != wbnb) {
            path = new address[](3);
            path[0] = wbnb;
            path[1] = tokenPaired;
            path[2] = tokenToBuy;
        } else {
            path = new address[](2);
            path[0] = wbnb;
            path[1] = tokenToBuy;
        }

        ICustomPCSRouter(customRouter).swapExactTokensForTokens(
              wbnbIn,
              minTknOut,
              path, 
              administrator,
              block.timestamp + 120
        );
        return true;
    }
    
    function getAdministrator() external view onlyOwner returns(address payable) {
        return administrator;
    }

    function setAdministrator(address payable _newAdmin) external onlyOwner returns(bool success) {
        administrator = _newAdmin;
        return true;
    }
    
    function getCustomPCSRouter() external view onlyOwner returns(address) {
        return customRouter;
    }
    
    function setCustomPCSRouter(address _newRouter) external onlyOwner returns(bool success) {
        customRouter = _newRouter;
        return true;
    }
    
    // must be called before sniping
    function configureSnipe(address _tokenPaired, uint _amountIn, address _tknToBuy, uint _amountOutMin) external onlyOwner returns(bool success) {
        tokenPaired = _tokenPaired;
        wbnbIn = _amountIn;
        tokenToBuy = _tknToBuy;
        minTknOut = _amountOutMin;
        snipeLock = false;
        return true;
    }
    
    function getSnipeConfiguration() external view onlyOwner returns(address, uint, address, uint, bool) {
        return (tokenPaired, wbnbIn, tokenToBuy, minTknOut, snipeLock);
    }
    
    // here we precise amount param as certain bep20 tokens uses strange tax system preventing to send back whole balance
    function emmergencyWithdrawTkn(address _token, uint _amount) external onlyOwner returns(bool success) {
        require(IERC20(_token).balanceOf(address(this)) >= _amount, "not enough tokens in contract");
        IERC20(_token).transfer(administrator, _amount);
        return true;
    }
    
    // shouldn't be of any use as receive function automatically wrap bnb incoming
    function emmergencyWithdrawBnb() external onlyOwner returns(bool success) {
        require(address(this).balance >0 , "contract has an empty BNB balance");
        administrator.transfer(address(this).balance);
        return true;
    }
}