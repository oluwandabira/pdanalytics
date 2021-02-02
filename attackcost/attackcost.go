package attackcost

import (
	"io"
	"net/http"
	"sync"

	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type Attackcost struct {
	client *dcrd.Dcrd
	server *web.Server
	xcBot  *exchanges.ExchangeBot

	height          int64
	hashrate        float64
	ticketPrice     float64
	ticketPoolSize  int64
	ticketPoolValue float64
	coinSupply      int64

	MeanVotingBlocks int64

	reorgLock sync.Mutex
}

func New(client *dcrd.Dcrd, webServer *web.Server, xcBot *exchanges.ExchangeBot) (*Attackcost, error) {
	ac := &Attackcost{
		server: webServer,
		xcBot:  xcBot,
		client: client,
	}

	hash, err := client.Rpc.GetBestBlockHash()
	if err != nil {
		return nil, err
	}
	blockHeader, err := client.Rpc.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	if err = ac.ConnectBlock(blockHeader); err != nil {
		return nil, err
	}

	ac.server.AddMenuItem(web.MenuItem{
		Href:      "/attack-cost",
		HyperText: "Attack Cost",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Decred Attack Cost",
		},
	})

	ac.server.Templates.AddTemplate("attackcost")

	ac.client.Notif.RegisterBlockHandlerGroup(ac.ConnectBlock)

	ac.server.AddRoute("/attack-cost", web.GET, ac.AttackCost)

	return ac, nil
}

func (ac *Attackcost) ConnectBlock(w *wire.BlockHeader) error {
	ac.reorgLock.Lock()
	defer ac.reorgLock.Unlock()
	ac.height = int64(w.Height)
	hash := w.BlockHash()

	// Hashrate
	header, err := ac.client.Rpc.GetBlockHeaderVerbose(&hash)
	if err != nil {
		return err
	}
	targetTimePerBlock := float64(ac.client.Params.TargetTimePerBlock)
	ac.hashrate = dcrd.CalculateHashRate(header.Difficulty, targetTimePerBlock)

	// Coin supply
	coinSupply, err := ac.client.Rpc.GetCoinSupply()
	if err != nil {
		return err
	}
	ac.coinSupply = int64(coinSupply)

	// Stake difficulty (ticket price)
	stakeDiff, err := ac.client.Rpc.GetStakeDifficulty()
	if err != nil {
		return err
	}
	ac.ticketPrice = stakeDiff.CurrentStakeDifficulty

	// Ticket pool info
	poolValue, err := ac.client.Rpc.GetTicketPoolValue()
	if err != nil {
		return err
	}
	ac.ticketPoolValue = poolValue.ToCoin()
	hashes, err := ac.client.Rpc.LiveTickets()
	if err != nil {
		return err
	}
	ac.ticketPoolSize = int64(len(hashes))

	return nil
}

// AttackCost is the page handler for the "/attack-cost" path.
func (ac *Attackcost) AttackCost(w http.ResponseWriter, r *http.Request) {
	price := 24.42
	if ac.xcBot != nil {
		if rate := ac.xcBot.Conversion(1.0); rate != nil {
			price = rate.Value
		}
	}

	ac.reorgLock.Lock()

	str, err := ac.server.Templates.ExecTemplateToString("attackcost", struct {
		*web.CommonPageData
		HashRate        float64
		Height          int64
		DCRPrice        float64
		TicketPrice     float64
		TicketPoolSize  int64
		TicketPoolValue float64
		CoinSupply      int64
	}{
		CommonPageData:  ac.server.CommonData(r),
		HashRate:        ac.hashrate,
		Height:          ac.height,
		DCRPrice:        price,
		TicketPrice:     ac.ticketPrice,
		TicketPoolSize:  ac.ticketPoolSize,
		TicketPoolValue: ac.ticketPoolValue,
		CoinSupply:      ac.coinSupply,
	})
	ac.reorgLock.Unlock()

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		ac.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}
