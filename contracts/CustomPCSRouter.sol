// SPDX-License-Identifier: MIT
pragma solidity >=0.6.0 <0.8.0;

import "@openzeppelin/contracts/math/SafeMath.sol";
import '@uniswap/v2-core/contracts/interfaces/IUniswapV2Pair.sol';
import "./TransferHelper.sol";
import "./pancake/PancakeLibrary.sol";

contract CustomPCSRouter {

    using SafeMath for uint;

    address public immutable factory = 0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73;
    address public immutable WETH = 0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c;

    modifier ensure(uint deadline) {
        require(deadline >= block.timestamp, 'PancakeRouter: EXPIRED');
        _;
    }

    receive() external payable {
        assert(msg.sender == WETH); // only accept ETH via fallback from the WETH contract
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

        for (uint i; i < path.length - 1; i++) {
            (address token0,) = PancakeLibrary.sortTokens(path[i], path[i + 1]);
            (uint amount0Out, uint amount1Out) = path[i] == token0 ? (uint(0), amounts[i + 1]) : (amounts[i + 1], uint(0));
            address addrTo = i < path.length - 2 ? PancakeLibrary.pairFor(factory, path[i + 1], path[i + 2]) : to;
            IUniswapV2Pair(PancakeLibrary.pairFor(factory, path[i], path[i + 1])).swap(
                amount0Out, amount1Out, addrTo, new bytes(0)
            );
        }
    }
}