# abe-miningpool-server

## Prerequisites



- A fully synchronized Abec node with some necessary APIs open
- A Abewallet node (optional)
- MySQL 5.7+ (You should prepare a user with write and read permission)



## Setup Pool Server

Test Environment: Ubuntu 20.04

### 1. Unzip

```shell
tar -xzvf tar -xzvf abe-miningpool-server-linux-amd64-v0.11.1.tar.gz
cd abe-miningpool-server-linux-amd64-v0.11.1
```

### 2. Modify the Configuration File

All the available config options are written in *miningpool-server.conf*. Please read them carefully and uncomment & modify the options which you want to add. Among all the options, the following must be configured (in order to connect to Abec and MySQL).

```
; dbaddress is the address of MySQL (default: 127.0.0.1:3306)
dbaddress=127.0.0.1:3306

; dbusername is the username which server used to connect with mysql, you must specify it.
dbusername=root

; dbpassword is the password which server used to connect with mysql, you must specify it.
dbpassword=root

; rpcconnect is the hostname/IP and port of Abec RPC server to connect to (default: localhost:8667)
rpcconnect=localhost:8667

; abecrpcuser and abecrpcpass are used as authentication to connect with abec, you must specify them.
abecrpcuser=rpcuser
abecrpcpass=rpcpass

; miningaddr specify the mining address to use when generating block template. You can add more than one mining address. Server will randomly pick one of them each time when generating block template.
miningaddr=a1b2c3d4......
miningaddr=a2212323......
```

If Abec needs TLS certificate to connect with, you should put the certificate into the working directory and rename it to *abec.cert* or specify the path of the certificate using option `cafile`.

If the pool server want to allocate the reward with abewallet, the following configuration items need to set.

```
; usewallet enable wallet to send transactions to miners automatically. (default: false).
; If this option is true, you also need to specify walletrpcuser, walletrpcpass, walletcafile (if needed), walletrpcconnect, servermanagefeeaddr
; walletrpcuser and walletrpcpass are used as authentication to connect with abewallet, you must specify them.
;walletrpcuser=rpcuse
;walletrpcpass=rpcpass

; walletcafile contains the path to the CA file of abewallet. If you do not specify it, server will use abewallet.cert in the working directory.
;walletcafile=./abewallet.cert

; walletrpcconnect is the hostname/IP and port of abewallet RPC server to connect to (default: localhost:8665)
;walletrpcconnect=127.0.0.1
```
### 3. Start

Finally start the server,

```shell
./abe-miningpool-server
```

You are required to enter the IP address to generate TLS certificate the first time you start the server,

```
Use config file: miningpool-server.conf
2022-09-21 17:04:32.991 [INF] POOL: Reward per blocks: 100
2022-09-21 17:04:33.020 [INF] POOL: Reward maturity: 50
2022-09-21 17:04:33.020 [INF] POOL: Manage fee: 20%
2022-09-21 17:04:33.021 [INF] POOL: Generating TLS certificates of mining pool...
Please enter the ip address of your server (press enter if you just test on local area network): 192.168.1.110
2022-09-21 17:04:39.411 [INF] POOL: Done generating TLS certificates
...
```

Then two files (*pool.cert* and *pool.key*) will be generated in the working directory. If you want to generate them again, delete these two files and restart the server. The file *pool.cert* should be given to the miners.

Then the server will create tables needed in the database. If it is the first time you start the mining pool server, it will also ask you to set the password for admin. This will create a special user named admin in database (in table user_infos). This username (admin) and password are used for administrator to connect to RPC server of mining pool server for advanced operations. See *mining_pool_api.md* for more information.

```
...
2022-09-21 17:14:21.359 [INF] POOL: Done generating TLS certificates
2022-09-21 17:14:21.359 [INF] POOL: Version 0.11.1
2022-09-21 17:14:21.360 [INF] DAL: Creating database abe_mining_pool...
2022-09-21 17:14:21.365 [INF] DAL: Creating table meta_infos...
2022-09-21 17:14:21.387 [INF] DAL: Creating table user_infos...
2022-09-21 17:14:21.398 [INF] DAL: Creating table detailed_share_infos...
2022-09-21 17:14:21.405 [INF] DAL: Creating table mined_block_infos...
2022-09-21 17:14:21.414 [INF] DAL: Creating table allocation_infos...
2022-09-21 17:14:21.418 [INF] DAL: Creating table user_share_infos...
2022-09-21 17:14:21.425 [INF] DAL: Connecting to database...
2022-09-21 17:14:21.426 [INF] DAL: Successfully connect to database
It seems that it is the first time you start the server, please specify the password for admin: 123456

```

After that, the mining pool will connect to Abec and Abewallet, if you see the following message, please enter your wallet private passphrase to unlock the connecting wallet for allocating the reward automatically. It means the mining pool is ready.
(Please ignore the log with **record not found**, server will automatically create it if the record is not found).

```
...
2022-09-21 18:35:29.832 [INF] POOL: Successfully generate account for admin, please use this password for RPC authentication

2022/09/21 18:35:29 [31;1mabe-miningpool-server/dal/dao/meta_info_dao.go:55 [35;1m record not found
[[33m[0.971ms] [34;1m[rows:0][0m SELECT * FROM `meta_infos` WHERE id = 1 ORDER BY `meta_infos`.`id` LIMIT 1
2022-09-21 18:35:29.835 [INF] POOL: Meta Info: Balance: 0, Last reward height: 0
Please specify the private passphrase for abewallet : 
2022-09-21 18:35:29.835 [INF] POOL: Attempting RPC client connection to Abec (localhost:8667)
2022-09-21 18:35:30.149 [INF] RPCCLIENT: Established connection to RPC server localhost:8667
2022-09-21 18:35:30.152 [INF] DAL: Disk storage enabled for ethash caches, CachesOnDisk = 3, dir = /home/user/.poolethash
2022-09-21 18:35:30.152 [INF] DAL: Disk storage enabled for ethash DAGs, DatasetsOnDisk = 3, dir = /home/user/.poolethash
2022-09-21 18:35:30.152 [INF] CHNS: Backend abec version: 0.11.1
2022-09-21 18:35:30.153 [INF] POSV: Pool RPC server listening on [::]:27777
2022-09-21 18:35:30.153 [INF] POSV: Pool RPC server listening on 0.0.0.0:27777
2022-09-21 18:35:30.177 [INF] CHNS: Get new block template from abec: {Version:536870912 Bits:1e01ffff CurTime:1663756530 Height:56000 PreviousHash:75c041d6a3842e8a9923b186f27de76412d6104a037fd1c74657d705915b5bc1 Transactions:[] CoinbaseTxn:0xc000484000}
2022-09-21 18:35:30.179 [DBG] MIMG: Miner manager receives a new block template from chain client, updating...
2022-09-21 18:35:30.180 [INF] MIMG: The coinbase reward of new template is 2560000000 Neutrino, Height 56000
2022-09-21 18:35:30.180 [INF] CHNS: Preparing verification cache for epoch 0
```



## API

We provide some HTTP-base JSON RPC APIs to get the information of the server. See *mining_pool_api.md* for details.



## Support Abelminer (GPU)

To support the protocol of Abelminer (GPU), you should open the TCP socket server by setting option `enabletcpsocket=true`. The server will listen on localhost:27778 in default, and this listening port can be modified by option `listenerporttcpsocket`. The TLS is on in default. To disable TLS, you can set option `notcpsockettls=true`.



## Tips

- **Automatic Difficulty Adjustment**: When a miner client connects with pool server, the pool server will distribute a task with default share difficulty. You do not need to worry about the share submit speed of the client because pool server will automatically adjust the share difficulty for each miner and record the share count according to the difficulty.
- **Reward Allocation Scheme**: Every few blocks (default 100 blocks, this value is controlled by option `rewardinterval`), the pool server will allocate rewards to miners according to their submitted share. And only those blocks that are mature will be allocated (default 200 blocks). For example, Suppose there are two mined blocks between block 56000 and 56099 (total reward 512ABE) and there are two miners who have submitted share between block 56000 and 56099 (Alice submits 10 share and Bob submits 30 share). The allocation will begin at height 56300 (namely 56100+200). The pool server will first deduct management fee from it. If the manage fee percent is 20, then 512*0.2=102.4 ABE is the management fee and 409.6 ABE is the reward that will be allocated to miners. Concretely, Alice will receive 102.4 ABE and Bob will receive 307.2 ABE. The details of allocation can be seen in table allocation_infos. If you change the allocation interval, it will take effect in the next epoch.
- **Mining Address**: You can omit the `miningaddr` option in *miningpool-server.conf*. If you do this, you should add option `useaddrfromabec=true`. Then the mining pool will use mining address from Abec. If you specify multiple mining addresses in configurations, the server will randomly pick one of them each time when generating block template.



## Database Description

database name: abe_mining_pool

## user_infos

This table stores the information of users (miners).

```mysql
CREATE TABLE `user_infos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(100) NOT NULL,
  `password` varchar(40) NOT NULL,
  `salt` varchar(20) NOT NULL,
  `email` varchar(100) DEFAULT NULL,
  `balance` bigint NOT NULL DEFAULT '0',
  `redeemed_balance` bigint NOT NULL DEFAULT '0',
  `address` longtext,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_idx_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```

The `username` in database is calculated by SHA256(SHA256(`address`)).

The `password` in database is calculated by (written in Golang),

```go
// Here password_plain is the password which is typed by user, salt is a random string which was generated when registering.
passwordHash := sha256.Sum256([]byte(password_plain + salt))
password := hex.EncodeToString(passwordHash[:])[:20]
```

- `balance` is the amount which a user owns (mining reward), **unit: Neutrino**
- `address` is the payment address.



## meta_infos

This table stores some meta info the mining pool.

```mysql
CREATE TABLE `meta_infos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `balance` bigint NOT NULL DEFAULT '0',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `last_reward_height` bigint NOT NULL DEFAULT '0',
  `last_reward_time` datetime(3) DEFAULT NULL,
  `allocated_balance` bigint NOT NULL DEFAULT '0',
  `last_tx_check_height` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```

- `balance` is the amount left (management fee) after allocating rewards to all miners. It does not include the amount that has not been allocated.
- `last_reward_height` means the reward mined before this height has been allocated to users.
- `allocated_balance` is the amount that has been allocated to users.
- `last_tx_check_height` means the height before that the transaction status has been confirmed, if there is an error or exception, they would be recorded in the log.


## allocation_infos

This table stores the allocation infos.

```mysql
CREATE TABLE `allocation_infos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(20) NOT NULL,
  `share_count` bigint NOT NULL DEFAULT '0',
  `balance` bigint NOT NULL DEFAULT '0',
  `start_height` bigint NOT NULL DEFAULT '0',
  `end_height` bigint NOT NULL DEFAULT '0',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```



## user_share_infos

This table stores the share count of each user in [start_height, end_height).

```mysql
CREATE TABLE `user_share_infos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(100) NOT NULL,
  `share_count` bigint NOT NULL DEFAULT '0',
  `start_height` bigint NOT NULL DEFAULT '0',
  `end_height` bigint NOT NULL DEFAULT '0',
  `allocated` bigint NOT NULL DEFAULT '0',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_share_infos_end_height` (`end_height`),
  KEY `idx_username` (`username`),
  KEY `idx_user_share_infos_start_height` (`start_height`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```



## detailed_share_infos

This table stores the detailed infos of all valid shares submitted  by users.

Note: The records in this table will only be created when `detailedshareinfo` option is on in order to avoid too much data.

```mysql
CREATE TABLE `share_infos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `share_count` bigint NOT NULL DEFAULT '0',
  `height` bigint NOT NULL DEFAULT '0',
  `share_hash` varchar(64) NOT NULL,
  `seal_hash` varchar(64) NOT NULL,
  `nonce` varchar(64) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_username` (`username`),
  KEY `idx_share_infos_height` (`height`),
  KEY `idx_share_infos_seal_hash` (`seal_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```



## mined_block_infos

This table stores the infos of mined blocks.

```mysql
CREATE TABLE `mined_block_infos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(20) NOT NULL,
  `height` bigint NOT NULL DEFAULT '0',
  `block_hash` varchar(64) NOT NULL,
  `seal_hash` varchar(64) NOT NULL,
  `reward` bigint unsigned NOT NULL DEFAULT '0',
  `disconnected` bigint NOT NULL DEFAULT '0',
  `connected` bigint NOT NULL DEFAULT '0',
  `rejected` bigint NOT NULL DEFAULT '0',
  `allocated` bigint NOT NULL DEFAULT '0',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `info` longtext,
  PRIMARY KEY (`id`),
  KEY `idx_username` (`username`),
  KEY `idx_mined_block_infos_block_hash` (`block_hash`),
  KEY `idx_mined_block_infos_height` (`height`),
  KEY `idx_mined_block_infos_seal_hash` (`seal_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```



## order_infos

This table stores the infos of transaction orders.

```mysql
CREATE TABLE `order_infos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `amount`  bigint default 0 NOT NULL,
  `status`  tinyint NOT NULL comment '0-transaction not sent, 1-transaction sent, 2-unable to send transactions (manual required)',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```



## order_transaction_infos

This table stores the infos of transactions. An order may correspond to several transactions

```mysql
CREATE TABLE `order_transaction_infos` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint NOT NULL,
  `transaction_hash` varchar(64) NOT NULL,
  `amount`  bigint DEFAULT 0 not null,
  `status` tinyint NOT NULL comment '0-not on chain,1-on chain,2-invalid',
  `height` bigint NOT NULL DEFAULT 0,
  `address` longtext NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_order_id` (`order_id`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```

