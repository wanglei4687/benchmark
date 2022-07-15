#!/usr/bin/env bash

export PULUMI_CONFIG_PASSPHRASE=$1
curl -s  $(pulumi stack output publicIp) | grep "done"
