package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	baseURL = "https://api.ntuity.io/v1/sites/%s/energy-flow/latest"
)

var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
var siteID = flag.String("site-id", "", "The ID of the site to collect metrics for")

type MetricValue struct {
	Value *float64  `json:"value"`
	Time  time.Time `json:"time"`
}

type EnergyFlow struct {
	PowerConsumption          MetricValue `json:"power_consumption"`
	PowerConsumptionCalc      MetricValue `json:"power_consumption_calc"`
	PowerProduction           MetricValue `json:"power_production"`
	PowerStorage              MetricValue `json:"power_storage"`
	PowerGrid                 MetricValue `json:"power_grid"`
	PowerChargingstations     MetricValue `json:"power_charging_stations"`
	PowerHeating              MetricValue `json:"power_heating"`
	PowerAppliances           MetricValue `json:"power_appliances"`
	StateOfCharge             MetricValue `json:"state_of_charge"`
	SelfSufficiency           MetricValue `json:"self_sufficiency"`
	ConsumersTotalCount       int         `json:"consumers_total_count"`
	ConsumersOnlineCount      int         `json:"consumers_online_count"`
	ProducersTotalCount       int         `json:"producers_total_count"`
	ProducersOnlineCount      int         `json:"producers_online_count"`
	StoragesTotalCount        int         `json:"storages_total_count"`
	StoragesOnlineCount       int         `json:"storages_online_count"`
	HeatingTotalCount         int         `json:"heatings_total_count"`
	HeatingsOnlineCount       int         `json:"heatings_online_count"`
	ChargingPointsTotalCount  int         `json:"charging_points_total_count"`
	ChargingPointsOnlineCount int         `json:"charging_points_online_count"`
	GirdsTotalCount           int         `json:"grids_total_count"`
	GridsOnlineCount          int         `json:"grids_online_count"`
}

func retrieveEnergyFlow(siteURL, apiKey string) (*EnergyFlow, error) {
	req, _ := http.NewRequest("GET", siteURL, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var flow EnergyFlow
	if err := json.Unmarshal(bs, &flow); err != nil {
		return nil, err
	}

	return &flow, nil
}

func startNtuityMetricsCollector(reg *prometheus.Registry) error {
	apiKey := os.Getenv("NTUITY_API_KEY")
	if len(apiKey) == 0 {
		return fmt.Errorf("no api key given")
	}

	siteURL := fmt.Sprintf(baseURL, *siteID)

	powerConsumption := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "power_consumption",
			Help:      "Power of all consumers, e.g. Appliances, CPs, HPs",
		},
		[]string{"site"},
	)

	powerConsumptionCalc := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "power_consumption_calc",
			Help:      "Calculated power of all consumers, e.g. Appliances, CPs, HPs",
		},
		[]string{"site"},
	)

	powerProduction := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "power_production",
			Help:      "Power of all producers, e.g. PVs",
		},
		[]string{"site"},
	)

	powerStorage := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "power_storage",
			Help:      "Power from + (=discharching) or to - (=charging) the storages",
		},
		[]string{"site"},
	)

	powerGrid := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "power_grid",
			Help:      "Power from + or to - the grid",
		},
		[]string{"site"},
	)

	powerChargingStations := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "power_charging_stations",
			Help:      "Power from + or to - the grid",
		},
		[]string{"site"},
	)

	powerHeating := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "power_heating",
			Help:      "Power of all heating devices",
		},
		[]string{"site"},
	)

	powerAppliances := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "power_appliances",
			Help:      "Power of all appliances (difference between total consumption and sum of all other sub-consumer)",
		},
		[]string{"site"},
	)

	stateOfCharge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "state_of_charge",
			Help:      "State of charge of all storages",
		},
		[]string{"site"},
	)

	selfSufficiency := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ntuity",
			Name:      "self_sufficiency",
			Help:      "A performance or fitness value about the current energy flow (based on power)",
		},
		[]string{"site"},
	)

	reg.MustRegister(
		powerConsumption,
		powerConsumptionCalc,
		powerProduction,
		powerStorage,
		powerGrid,
		powerChargingStations,
		powerHeating,
		powerAppliances,
		stateOfCharge,
		selfSufficiency)

	go func() {
		for {
			flow, err := retrieveEnergyFlow(siteURL, apiKey)
			if err != nil {
				log.Printf("Failed to collect metrics: %v", err)
				os.Exit(1)
			}

			if flow.PowerConsumptionCalc.Value != nil {
				powerConsumptionCalc.WithLabelValues(*siteID).Set(float64(*flow.PowerConsumptionCalc.Value))
			} else {
				powerConsumptionCalc.WithLabelValues(*siteID).Set(float64(0))
			}
			if flow.PowerProduction.Value != nil {
				powerProduction.WithLabelValues(*siteID).Set(float64(*flow.PowerProduction.Value))
			} else {
				powerProduction.WithLabelValues(*siteID).Set(float64(0))
			}
			if flow.PowerStorage.Value != nil {
				powerStorage.WithLabelValues(*siteID).Set(float64(*flow.PowerStorage.Value))
			} else {
				powerStorage.WithLabelValues(*siteID).Set(float64(0))
			}
			if flow.PowerGrid.Value != nil {
				powerGrid.WithLabelValues(*siteID).Set(float64(*flow.PowerGrid.Value))
			} else {
				powerGrid.WithLabelValues(*siteID).Set(float64(0))
			}
			if flow.PowerChargingstations.Value != nil {
				powerChargingStations.WithLabelValues(*siteID).Set(float64(*flow.PowerChargingstations.Value))
			} else {
				powerChargingStations.WithLabelValues(*siteID).Set(float64(0))
			}
			if flow.PowerHeating.Value != nil {
				powerHeating.WithLabelValues(*siteID).Set(float64(*flow.PowerHeating.Value))
			} else {
				powerHeating.WithLabelValues(*siteID).Set(float64(0))
			}
			if flow.PowerAppliances.Value != nil {
				powerAppliances.WithLabelValues(*siteID).Set(float64(*flow.PowerAppliances.Value))
			} else {
				powerAppliances.WithLabelValues(*siteID).Set(float64(0))
			}
			if flow.StateOfCharge.Value != nil {
				stateOfCharge.WithLabelValues(*siteID).Set(float64(*flow.StateOfCharge.Value))
			} else {
				stateOfCharge.WithLabelValues(*siteID).Set(float64(0))
			}
			if flow.SelfSufficiency.Value != nil {
				selfSufficiency.WithLabelValues(*siteID).Set(float64(*flow.SelfSufficiency.Value))
			} else {
				selfSufficiency.WithLabelValues(*siteID).Set(float64(0))
			}

			time.Sleep(time.Second * 60)
		}
	}()

	return nil
}

func main() {
	flag.Parse()

	if len(*siteID) == 0 {
		log.Printf("No site ID given")
		os.Exit(1)
	}

	reg := prometheus.NewRegistry()

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))

	log.Printf("Listening on %s", *addr)

	if err := startNtuityMetricsCollector(reg); err != nil {
		log.Printf("Failed to start metrics collector: %v", err)
		os.Exit(1)
	}

	log.Fatal(http.ListenAndServe(*addr, nil))
}
