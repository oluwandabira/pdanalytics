package main

import (
	"fmt"

	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/attackcost"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/homepage"
	"github.com/planetdecred/pdanalytics/parameters"
	"github.com/planetdecred/pdanalytics/stakingreward"
	"github.com/planetdecred/pdanalytics/web"
)

func setupModules(cfg *config, client *dcrd.Dcrd, server *web.Server, xcBot *exchanges.ExchangeBot) error {
	var err error

	var stk *stakingreward.Calculator
	if cfg.EnableStakingRewardCalculator {
		stk, err = stakingreward.New(client, server, xcBot)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create staking reward component, %s", err.Error())
		}

		log.Info("Staking Reward Calculator Enabled")
	}

	var prms *parameters.Parameters
	if cfg.EnableChainParameters {
		prms, err = parameters.New(client, server)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create parameters component, %s", err.Error())
		}

		log.Info("Chain Parameters Enabled")
	}

	var ac *attackcost.Attackcost
	if cfg.EnableAttackCost {
		ac, err = attackcost.New(client, server, xcBot)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create attackcost component, %s", err.Error())
		}

		log.Info("Attack Cost Calculator Enabled")
	}

	_, err = homepage.New(server, homepage.Mods{
		Stk: stk,
		Prm: prms,
		Ac:  ac,
	})
	if err != nil {
		log.Error(err)
	}

	return nil
}
