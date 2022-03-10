package main

import (
	"controller/pkg/util/conversion"
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Link struct {
	ip   string
	mac  string
	port uint32
}

type LinkBytes struct {
	ip   []byte
	mac  []byte
	port []byte
}

type host struct {
	name string
	Ip   string
	Mac  string
}

func (link *Link) toBytes() LinkBytes {
	ip, _ := conversion.IpToBinary(link.ip)
	mac, _ := conversion.MacToBinary(link.mac)
	port, _ := conversion.UInt32ToBinaryCompressed(link.port)
	return LinkBytes{
		ip:   ip,
		mac:  mac,
		port: port,
	}
}

func parseHosts(fileName string) []host {
	// Open our jsonFile
	jsonFile, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()
	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	var topo struct {
		Hosts map[string]host
	}
	if err = json.Unmarshal(jsonBytes, &topo); err != nil {
		log.Fatal(err)
	}
	hosts := make([]host, 0, len(topo.Hosts))
	for key, val := range topo.Hosts {
		hosts = append(hosts, host{
			name: key,
			Ip:   val.Ip,
			Mac:  val.Mac,
		})
	}
	return hosts
}

func parseRoutes(fileName string, swName string) []string {
	// Open our jsonFile
	jsonFile, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()
	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	var routes map[string][]string
	if err = json.Unmarshal(jsonBytes, &routes); err != nil {
		log.Fatal(err)
	}
	return routes[swName]
}

func GetLinks(id uint64, topoFile string, routeFile string) []Link {
	swName := "s" + strconv.FormatUint(id, 10)
	hosts := parseHosts(topoFile)
	routes := parseRoutes(routeFile, swName)
	links := make([]Link, 0, 4)
	// foreach link
	for idx, route := range routes {
		// find the host
		for _, host := range hosts {
			if host.name == route {
				links = append(links, Link{
					ip:   strings.Split(host.Ip, "/")[0],
					mac:  host.Mac,
					port: uint32(idx + 1),
				})
			}
		}
	}
	return links
}

func GetDefaultLinks(id uint64) []Link {
	return GetLinks(id, "../mininet/topology.json", "routes.json")
}

func GetLinksBytes(links []Link) []LinkBytes {
	linksByte := make([]LinkBytes, len(links))
	for idx, link := range links {
		linksByte[idx] = link.toBytes()
	}
	return linksByte
}
