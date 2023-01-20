# ntuity-collector

This provides a Go based prometheus scrape endpoint for metrics provided by [ntuity.io](https://ntuity.io/) around energy flow and cusumption in different sites.

The implementation uses the [official API](https://docs.ntuity.io/docs)

## Build

    go build -o collector ./cmd/ntuity-collector


## Run

To run the service, simply run

    NTUITY_API_KEY=<your API key> ./collector -site <your site id>

Afterwards you can scrape metrics via Prometheus from https://127.0.0.1:8080/metrics
