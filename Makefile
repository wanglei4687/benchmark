MAKEFLAGS += --warn-undefined-variables --no-builtin-rules
SHELL := /usr/bin/env bash
.SHELLFLAGS := -uo pipefail -c
.DEFAULT_GOAL := help
.DELETE_ON_ERROR:
.SUFFIXES:

PASSPHRASE?=123

export PULUMI_CONFIG_PASSPHRASE=$(PASSPHRASE)

.PHONY: config
config:
	pulumi config set --path 'config.az' us-east-1a
	pulumi config set --path 'config.instance' t2.micro
	pulumi config set --path 'config.region' us-east-1
	pulumi config set --path 'config.ami' ami-0cff7528ff583bf9a
	pulumi config set --path 'config.capacitystatus' UnusedCapacityReservation
	pulumi config set --path 'config.instancesku'  S948KKM542ZP8Y37
	pulumi config set --path 'config.vol[0].multiattach' true
	pulumi config set --path 'config.vol[0].type' gp2
	pulumi config set --path 'config.vol[0].size' 10
	pulumi config set --path 'config.vol[0].devicename' /dev/sdc
	pulumi config set --path 'config.vol[0].version'  'General Purpose'

.PHONY: stack
stack:
	pulumi stack

.PHONY: preview
preview:
	pulumi preview


.PHONY: up
up:
	pulumi up --yes


.PHONY: down
down:
	pulumi down --yes


.PHONY: show
show:
	pulumi stack output
