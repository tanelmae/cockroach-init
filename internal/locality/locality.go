package locality

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

/*
	Provides locality for the CockroachDB node based on GCP metadata.
	Produces file <confDir>/locality

	CRDB locality would have 4 levels:
	provider={gcp,yandex, ...} // cloud provider
	area={na, sa, eu, oc, as, ...} // rough geo location
	territory={eu, us, ru ...} // legal location
	zone={europe-north1-a ...} // cloud provider datacenter zone

	It can also discover other CRDB nodes from SRV records.
*/

var (
	debug *bool
	/*  Converts GKE cluster location to ISO 3166 country codes
	    This can be used to satisfy any legal regulation
		about data storage location.
	*/
	gcpTerritories = map[string]string{
		"asia-south1":             "IN",
		"asia-southeast1":         "SG",
		"asia-east2":              "CN",
		"asia-east1":              "TW",
		"asia-northeast1":         "JP",
		"asia-northeast2":         "JP",
		"australia-southeast1":    "AU",
		"europe-west2":            "GB",
		"europe-west1":            "BA",
		"europe-west4":            "NL",
		"europe-west6":            "CH",
		"europe-west3":            "DE",
		"europe-north1":           "FI",
		"us-west1":                "US",
		"us-west2":                "US",
		"us-central1":             "US",
		"us-east1":                "US",
		"us-east4":                "US",
		"northamerica-northeast1": "CA",
		"southamerica-east1":      "BR",
	}

	// Maps GCP areas to continent codes
	// This is for geo proximity.
	gcpAreas = map[string]string{
		"asia":         "AS",
		"europe":       "EU",
		"northamerica": "NA",
		"us":           "NA",
		"southamerica": "SA",
		"australia":    "OC",
	}
)

// Locality hold CockroachDB locality data
// https://www.cockroachlabs.com/docs/stable/training/locality-and-replication-zones.html
type Locality struct {
	Provider  string // cloud provider
	Area      string // rough geo location
	Territory string // ISO 3166 code for location
	Zone      string // cloud provider datacenter zone full name
}

// Provides the correct format for CRDB
func (l *Locality) String() string {
	return fmt.Sprintf("provider=%s,area=%s,territory=%s,location=%s", l.Provider, l.Area, l.Territory, l.Zone)
}

// FromMetadata tries to resolve CockroachDB node locality from
// cloud provider instance metadata
func FromMetadata() (*Locality, error) {

	// Current implementation only supports GCP
	ips, _ := net.LookupIP("metadata.google.internam")
	if ips == nil {
		return nil, errors.New("Not running on GCP")
	}
	clusterName := getGCPMetadata("/instance/attributes/cluster-name")
	if *debug {
		log.Println("cluster name: ", clusterName)
	}

	clusterLocation := getGCPMetadata("/instance/attributes/cluster-location")
	if *debug {
		log.Println("cluster location: ", clusterLocation)
	}

	zone := getGCPNodeZone()

	firstDelimiter := strings.Index(clusterLocation, "-")
	secondDelimiter := strings.LastIndex(clusterLocation, "-")

	// Sample zone asia-southeast1-a -> AS
	area := gcpAreas[clusterLocation[:firstDelimiter]]
	// asia-southeast1 -> SG
	territory := gcpTerritories[clusterLocation[:secondDelimiter]]

	return &Locality{
		Provider:  "gcp",
		Area:      area,
		Territory: territory,
		Zone:      zone,
	}, nil
}

func getGCPMetadata(urlPath string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://metadata/computeMetadata/v1"+urlPath, nil)
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(bodyBytes)
}

func getGCPNodeZone() string {
	zoneResponse := getGCPMetadata("/instance/zone")
	// We just need the last part of the response
	// projects/896444227315/zones/europe-north1-a -> europe-north1-a

	gkeZoneSlice := strings.Split(zoneResponse, "/")
	return gkeZoneSlice[len(gkeZoneSlice)-1]
}
