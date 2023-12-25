This module is to automatically create transactions to distribute all miners' balances.

In general, once a miner has a balance that needs to be distributed, it should complete the distribution by creating a
transaction. But taking into account the eventual consistency of the chain, and in order to reduce the transaction fee
that needs to be provided for packaging transaction into the chain (assuming that the transaction fee is borne by the
mining pool), increase the profitable part of the mining pool.

By setting a threshold, when the miner's balance reaches this threshold, a transaction is created. Inevitably, this will
cause some delay, but will not cause any economic loss.

In order to work with other modules, appropriate distribution delays will be introduced in this module.
In order to adjust with the block reward, the automatic adjustment threshold of the percentage is adopted.

Based on block preset rewards, after setting a management fee ratio, the remain rewards will be divided into four units.
When the miner balance reaches one/two/four units respectively, the corresponding amount order will be created, and the
composable orders should be processed through a transaction.

In order to combine orders with different amounts, one unspent transaction output (UTXO for short) will have four units,
and there are four combinations that can be combined, namely [4],[2,2],[2,1,1],[1,1,1,1], the priority is
[4]>[2,2]>[1,1,1,1]>[2,1,1].

After successful combination, assign a UTXO to each order in that combination, at the same time the order is marked as
successful. To binding the orders with user information, it create a record can be identified by request_hash field.

The request_hash field is used to combine relevant order information, initiate a transaction request, create a
transaction, and track the status by transaction hash.

The above process is triggered periodically, and at the same time, another function that tracks transaction status is
also triggered periodically， in addition, the transaction status is also updated with notification.

In summary, the entire process of automatically distributing balances can be divided into multiple stages: 
- Create orders based on redeemable balances and allocatable units
- Combine order and assign utxo, create record to track status
- Receive notifications and track status regularly