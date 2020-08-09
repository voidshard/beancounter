/*Basic command structure*/
package main

import (
	"github.com/alecthomas/kong"
)

// context holds global options
type context struct{}

// cli commands / args available
var cli struct {
	Ctx context `embed`

	Link linkCmd `cmd help:"Link a bank to beancounter."`
}

type linkCmd struct {
	Truelayer truelayerCmd `cmd help:"Use truelayer as the provider"`
}

type truelayerCmd struct {
	Port              int    `help:"Port to host HTTP server on (listens for Truelayer message)." default:8500`
	Redirect          string `required help:"URL to have Truelayer send OAuth response to."`
	TruelayerClientId string `name:"client-id" required help:"Truelayer client ID."`
	TruelayerSecret   string `name:"secret" required help:"Truelayer client secret."`
	Days              int    `default:1095 help:"Number of days backward to fetch transactions."`
	Out               string `default:"jsonfile:out.json" help:"Where to write [jsonfile:/path/file.json es8:http://myelasticsearch:9200]"`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&cli.Ctx)
	ctx.FatalIfErrorf(err)
}
