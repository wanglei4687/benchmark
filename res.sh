#!/usr/bin/env bash

export PULUMI_CONFIG_PASSPHRASE=$1
pulumi stack output >> result.txt
echo "---------------------------------------------------" >> result.txt
curl $(pulumi stack output publicIp) >> result.txt
