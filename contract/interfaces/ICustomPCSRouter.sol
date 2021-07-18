// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface ICustomPCSRouter {
    function swapExactTokensForTokens(
        uint amountIn,
        uint amountOutMin,
        address[] calldata path,
        address to,
        uint deadline
    ) external virtual ensure(deadline) returns (uint[] memory amounts);
}