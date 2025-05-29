#!/bin/sh

set -e

C1V1_HOME="$HOME/.sourcehub-1-1"
C2V1_HOME="$HOME/.sourcehub-2-1"
C1V2_HOME="$HOME/.sourcehub-1-2"
C2V2_HOME="$HOME/.sourcehub-2-2"
HERMES_HOME="$HOME/.hermes"

C1V2_JSON="scripts/validator1-2.json"
C2V2_JSON="scripts/validator2-2.json"

ps aux | grep sourcehubd

killall sourcehubd 2>/dev/null || true
killall hermes 2>/dev/null || true

sleep 1

rm -rf $C1V1_HOME
rm -rf $C2V1_HOME
rm -rf $HERMES_HOME
rm -rf $C1V2_HOME
rm -rf $C2V2_HOME
rm -rf chain_*.log
rm -rf hermes.log
rm $C1V2_JSON 2>/dev/null || true
rm $C2V2_JSON 2>/dev/null || true

echo "DONE"
