// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.6.0 <0.8.0;

import "@openzeppelin/contracts/math/SafeMath.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import '@uniswap/v2-core/contracts/interfaces/IUniswapV2Pair.sol';
import "./TransferHelper.sol";
import "./uniswap/UniSwapV2Library.sol";

contract CustomRouter is Ownable {

    using SafeMath for uint;

    address private factory;
    address private wbnb;

    bytes32 private creationCode;

    constructor(address _factory, address _wbnb, bytes32 _creationCode) public {
        factory = _factory;
        wbnb = _wbnb;
        creationCode = _creationCode;
    }

    modifier ensure(uint deadline) {
        require(deadline >= block.timestamp, 'Router: EXPIRED');
        _;
    }

    receive() external payable {
        assert(msg.sender == wbnb); // only accept BNB via fallback from the wbnb contract
    }

    function setFactoryAddress(address _factory, bytes32 _creationCode) external onlyOwner returns(bool success) {
        factory = _factory;
        creationCode = _creationCode; // creationCode changes too.
        return true;
    }

    function getFactoryAddress() external view onlyOwner returns(address) {
        return factory;
    }

    function setWBNBAddress(address _wbnb) external onlyOwner returns(bool success) {
        wbnb = _wbnb;
        return true;
    }

    function getWBNBAddress() external view onlyOwner returns(address) {
        return wbnb;
    }

    function getCreationCode() external view onlyOwner returns(bytes32) {
        return creationCode;
    }

    function swapExactTokensForTokens(
        uint amountIn,
        uint amountOutMin,
        address[] calldata path,
        address to,
        uint deadline
    ) external virtual ensure(deadline) returns (uint[] memory amounts) {
        amounts = UniSwapV2Library.getAmountsOut(factory, amountIn, path, creationCode);
        require(amounts[amounts.length - 1] >= amountOutMin, 'Router: INSUFFICIENT_OUTPUT_AMOUNT');
        TransferHelper.safeTransferFrom(
            path[0], msg.sender, UniSwapV2Library.pairFor(factory, path[0], path[1], creationCode), amounts[0]
        );

        route(amounts, path, to);
    }

    function route(uint[] memory amounts, address[] memory path, address _to) private {
        for (uint i; i < path.length - 1; i++) {
            (address input, address output) = (path[i], path[i + 1]);
            (address token0,) = UniSwapV2Library.sortTokens(input, output);
            uint amountOut = amounts[i + 1];
            (uint amount0Out, uint amount1Out) = input == token0 ? (uint(0), amountOut) : (amountOut, uint(0));
            address to = i < path.length - 2 ? UniSwapV2Library.pairFor(factory, output, path[i + 2], creationCode) : _to;
            IUniswapV2Pair(UniSwapV2Library.pairFor(factory, input, output, creationCode)).swap(
                amount0Out, amount1Out, to, new bytes(0)
            );
        }
    }
}