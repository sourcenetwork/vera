#!/bin/bash

CHAINID=sourcehub
CMD=sourcehubd

# Get account address from account name.
sh-acc-addr() {
    local account_name=$1
    local address=$($CMD keys show "$account_name" --output json | jq -r '.address')

    if [ -z "$address" ]; then
        echo "Error: Could not find address for account name '$account_name'"
        return 1
    else
        echo "$address"
        return 0
    fi
}

sh-val-addr() {
    validator_name=$1
    if [ -z "$validator_name" ]; then
        $CMD q staking validators --output json | jq -r '.validators[0].operator_address'
    else
        $CMD q staking validator $validator_name --output json | jq -r '.validator.operator_address'
    fi
}

sh-mod-addr() {
    mod_name=$1
    $CMD q auth module-account $mod_name -o json | jq -r .account.value.address
}

sh-balances() {
    account_list=$*
    if [ -z "$account_list" ]; then
        account_list=`$CMD keys list --output json | jq -r '.[].name'`
    fi
    for account_name in $account_list; do
        echo $account_name;
        if [[ $account_name == mod_* ]]; then
            mod_name=${account_name:4}
            account_name=`sh-mod-addr $mod_name`
        fi
        $CMD q bank balances $account_name --output json | jq '.balances';
    done
}

sh-send() {
    from=$1
    to=$2
    amount=$3
    $CMD tx bank send `sh-acc-addr $from` `sh-acc-addr $to` $amount -y --chain-id $CHAINID --gas auto
}

sh-stake() {
    validator_name=$1
    from=$2
    amount=$3
    $CMD tx staking delegate `sh-val-addr $validator_name` $amount --from $from -y --chain-id $CHAINID --gas auto
}

sh-lock() {
    validator_name=$1
    from=$2
    amount=$3
    $CMD tx tier lock `sh-val-addr $validator_name` $amount --from $from -y --chain-id $CHAINID --gas auto
}

sh-unlock() {
    validator_name=$1
    from=$2
    amount=$3
    $CMD tx tier unlock `sh-val-addr $validator_name` $amount --from $from -y --chain-id $CHAINID --gas auto
}

sh-cancel-unlocking() {
    validator_name=$1
    from=$2
    amount=$3
    creation_height=$4
    $CMD tx tier cancel-unlocking `sh-val-addr $validator_name` $amount $creation_height --from $from -y --chain-id $CHAINID --gas auto
}
