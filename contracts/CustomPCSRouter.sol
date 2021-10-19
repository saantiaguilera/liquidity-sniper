// SPDX-License-Identifier: MIT
pragma solidity >=0.6.0 <0.8.0;

import "@openzeppelin/contracts/math/SafeMath.sol";
import '@uniswap/v2-core/contracts/interfaces/IUniswapV2Pair.sol';
import "./TransferHelper.sol";
import "./pancake/PancakeLibrary.sol";

contract CustomPCSRouter {

    using SafeMath for uint;

    // mainnet values
    address public immutable factory = 0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73;
    address public immutable wbnb = 0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c;

    // testnet values
    // address public immutable factory = 0x6725F303b657a9451d8BA641348b6761A6CC7a17;
    // address public immutable wbnb = 0xae13d989daC2f0dEbFf460aC112a837C89BAa7cd;

    modifier ensure(uint deadline) {
        require(deadline >= block.timestamp, 'PancakeRouter: EXPIRED');
        _;
    }

    receive() external payable {
        assert(msg.sender == wbnb); // only accept BNB via fallback from the wbnb contract
    }

    function swapExactTokensForTokens(
        uint amountIn,
        uint amountOutMin,
        address[] calldata path,
        address to,
        uint deadline
    ) external virtual ensure(deadline) returns (uint[] memory amounts) {
        amounts = PancakeLibrary.getAmountsOut(factory, amountIn, path);
        require(amounts[amounts.length - 1] >= amountOutMin, 'PancakeRouter: INSUFFICIENT_OUTPUT_AMOUNT');
        TransferHelper.safeTransferFrom(
            path[0], msg.sender, PancakeLibrary.pairFor(factory, path[0], path[1]), amounts[0]
        );

        route(amounts, path, to);
    }

    function route(uint[] memory amounts, address[] memory path, address _to) private {
        for (uint i; i < path.length - 1; i++) {
            (address input, address output) = (path[i], path[i + 1]);
            (address token0,) = PancakeLibrary.sortTokens(input, output);
            uint amountOut = amounts[i + 1];
            (uint amount0Out, uint amount1Out) = input == token0 ? (uint(0), amountOut) : (amountOut, uint(0));
            address to = i < path.length - 2 ? PancakeLibrary.pairFor(factory, output, path[i + 2]) : _to;
            IUniswapV2Pair(PancakeLibrary.pairFor(factory, input, output)).swap(
                amount0Out, amount1Out, to, new bytes(0)
            );
        }
    }
}