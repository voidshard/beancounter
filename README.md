# Beancounter

Simple tool to fetch transaction data from banks for local storage / processing / visualization.


## Requirements

Required:

[golang](https://golang.org/dl/) 
- Obviously

[ngrok](https://ngrok.com/download) 
- Allows us to receive OAuth redirects to DNS resolvable address (free account is fine). (If you own a domain & can add DNS records you could use that instead).

Helpful:

[docker](https://docs.docker.com/get-docker/) & [docker-compose](https://docs.docker.com/compose/install/)
- A docker-compose file is included to easily stand up a local ElasticSearch & Kibana for making visualizations


## Building

Nothing special here, it's vanilla Go with only a handful of depends

```bash
> git clone http://github.com/voidshard/beancounter
> cd beancounter
> go build -o beancounter cmd/beancounter/*.go
```
This is using Go modules, so you'll want a Go version more recent than [1.11](https://blog.golang.org/using-go-modules)


## Getting Transactions

Rather than trying to support every bank since ever, we lean on a data provider to collect the info for us. In doing so we ask for read-only, non-renewable, shortlived token(s) & limited scopes.


### Truelayer

Currently the only data provider supported is [Truelayer](https://truelayer.com/).

In order to use this, you'll need some valid Truelayer credentials and to whitelist some DNS resolvable URI so that Truelayer can message you back after the OAuth flow.


Basic steps:
- Start up ngrok using 
```bash
ngrok http 8500
```
you should see a link like "https://xxxxx.ngrok.io" (the "redirect" URL), note this down.
- Sign up for an account with Truelayer
- Open up console.truelayer.com
- Make sure the box at the top is "Live" not "Sandbox"
- Note down your app "client_id" and "client_secret"
- Under "Allowed redirect URIs" add your ngrok https redirect URL
- Check again that the URL output from ngrok & the allowed URL are the same 
- (optional) If using docker, run "docker-compose up" to stand up ElasticSearch & Kibana locally
- Run the tool, inserting your values
```bash
./beancounter link truelayer --redirect URL --client-id ID --secret SECRET 
```
- Open the printed link in a browser & follow the instructions (the steps for each bank are different)
```
Go to: https://auth.truelayer.com/?......
```
- Once the flow is completed a temporary code will be sent to the tool (via ngrok) and the tool will resume

Note: If you're using a free ngrok account the redirect URL from ngrok will change when you restart it. This is ok as you can always reopen the truelayer console & set it to whatever ngrok is currently using. Just know that this will *not* work if the URL isn't in truelayer's whitelist.

For the security minded, we ask for the [scopes](https://docs.truelayer.com/) (you can see this encoded in the printed link auth.truelayer.com)
- balance
- transactions
- accounts

We also add an ecrypted signed state that we check for on the redirect message (the encryption & signing keys are randomly generated each run).


- By default this pulls 3 years worth of transactions from today. You can pull more or less as you wish.
- You can pull data from as many banks as you like this way, the tool includes the bank/account name on each transaction.


## Saving Output

At the moment by default the tool outputs json to a file "out.json". You can write to a file or index transactions straight into ElasticSearch. An output is specified via type:path. Eg a json file "/tmp/foobar.json" would be "--out jsonfile:/tmp/foobar.json". An ElasticSearch listening on localhost:9200 would be "--out es8:http://localhost:9200"

This tool doesn't attempt to do any postprocessing of the data it gets, depend on which bank(s) you're linking to you may or may not want to clean it up / standardize it.

