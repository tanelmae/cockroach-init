package discovery

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"strings"
)

// FindNodes takes list of SRV records and outputs list of reachable nodes
func FindNodes(srv []string, max int) string {
	var srvs []*net.SRV

	// Resolve all SRV endpoints, merge together and sort by priority
	for _, s := range srv {
		endpoints, err := resolveSRVRecords(s)
		if err != nil {
			log.Fatalln(err)
		}
		srvs = append(srvs, endpoints...)
	}
	byPriorityWeight(srvs).sort()

	// Check if found endpoints are reachable.
	// Will confirm that domain can be resolved and TCP
	// connection can be created.
	// Once it has the max reachable endpoints it is done.
	var reachable []string
	for _, s := range srvs {
		endpoint := fmt.Sprintf("%s:%d", s.Target, s.Port)
		conn, err := net.Dial("tcp", endpoint)
		if err != nil {
			log.Println("could not connect to server: ", err)
		} else {
			reachable = append(reachable, endpoint)
		}
		conn.Close()

		if len(reachable) == max {
			break
		}
	}

	return strings.Join(reachable, ",")
}

// NewSRVRecord takes a name of SRV record and tries to parse it
// _port._proto.example.com.
// SRV record "service" part is called port here
// as on Kubernetes it is the name of the named port.
func resolveSRVRecords(raw string) ([]*net.SRV, error) {
	if !strings.HasPrefix(raw, "_") {
		return nil, fmt.Errorf("SRV record should start with _: %s", raw)
	}

	// This is bit stupid as under the hood net lib will redo it
	sp := strings.Split(raw, ".")
	if len(sp) < 3 {
		return nil, fmt.Errorf("not a SRV record: %s", raw)
	}

	port := strings.TrimPrefix(sp[0], "_")
	proto := strings.TrimPrefix(sp[1], "_")

	name := strings.TrimPrefix(raw, fmt.Sprintf("_%s._%s.", port, proto))

	_, srvs, err := net.LookupSRV(port, proto, name)
	if err != nil {
		return nil, err
	}
	return srvs, nil
}

// Copied from the net lib
// byPriorityWeight sorts SRV records by ascending priority and weight.
type byPriorityWeight []*net.SRV

func (s byPriorityWeight) Len() int {
	return len(s)
}

func (s byPriorityWeight) Less(i, j int) bool {
	return s[i].Priority < s[j].Priority || (s[i].Priority == s[j].Priority && s[i].Weight < s[j].Weight)
}

func (s byPriorityWeight) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// shuffleByWeight shuffles SRV records by weight using the algorithm
// described in RFC 2782.
func (addrs byPriorityWeight) shuffleByWeight() {
	sum := 0
	for _, addr := range addrs {
		sum += int(addr.Weight)
	}
	for sum > 0 && len(addrs) > 1 {
		s := 0
		n := rand.Intn(sum)
		for i := range addrs {
			s += int(addrs[i].Weight)
			if s > n {
				if i > 0 {
					addrs[0], addrs[i] = addrs[i], addrs[0]
				}
				break
			}
		}
		sum -= int(addrs[0].Weight)
		addrs = addrs[1:]
	}
}

// sort reorders SRV records as specified in RFC 2782.
func (addrs byPriorityWeight) sort() {
	sort.Sort(addrs)
	i := 0
	for j := 1; j < len(addrs); j++ {
		if addrs[i].Priority != addrs[j].Priority {
			addrs[i:j].shuffleByWeight()
			i = j
		}
	}
	addrs[i:].shuffleByWeight()
}
