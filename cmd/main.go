package main

import (
	"flag"
	"fmt"
	"github.com/tanelmae/cockroach-init/internal/config"
	"github.com/tanelmae/cockroach-init/internal/discovery"
	"github.com/tanelmae/cockroach-init/internal/locality"
	"log"
	"os"
)

// Version of this service
var Version = "dev"

/*
	Generates CockroachDB runner script.
	Assumes that /bin/sh is available but doesn't expect any other commands than builtin ones.
	Script looks like:
		#!/bin/sh
		export COCKROACH_CHANNEL=kubernetes-secure
		/cockroach/cockroach start <resolved args>
	Args can be passed as YAML file. Any key/value pair in the YAML will be passed to the cockroach binary as an argument.
	Node locality can be dynamically resolved from cloud provider instance metadata and join list can be resolved from SRV DNS records.
*/

func main() {
	log.Printf("Version: %s\n", Version)
	debug := flag.Bool("debug", false, "Debug mode")
	autoLocality := flag.Bool("auto-locality", false, "Enables resolving locality from instance metadata")
	srvDisco := flag.Bool("service-discovery", false, "Enables service discovery based on SRV records")

	output := flag.String("output", "/tmp/crdb-start.sh", "Path to startup scrit to be generated")
	configFile := flag.String("config", "", "Path to config file")

	flag.Parse()

	if *configFile == "" {
		log.Println("No config file. Use the -config flag.")
		os.Exit(1)
	}

	c, err := config.Read(*configFile)
	if err != nil {
		log.Fatalln(err)
	}

	if *debug {
		log.Printf("Output path: %s", *output)
	}

	if *autoLocality {
		locality, err := locality.FromMetadata()
		if err != nil {
			log.Fatalln(err)
		}
		log.Println(*locality)
		c.SetLocality((*locality).String())
	}

	if *srvDisco {
		join := discovery.FindNodes(c.SRV, c.JoinMax)
		if join != "" {
			c.SetJoin(join)
		} else {
			log.Println("No CockroachDB nodes discovered")
			log.Printf("SRVs used for lookup: %s\n", c.SRV)
		}
	}

	runnerCmd := fmt.Sprintf("#!/bin/sh\nexport COCKROACH_CHANNEL=kubernetes-secure\n%s", c.ExecCmd())

	if *debug {
		log.Printf("Generated runner:\n%s\n", runnerCmd)
	}

	f, err := os.Create(*output)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	f.WriteString(runnerCmd)
	f.Chmod(os.FileMode(0755))
}
