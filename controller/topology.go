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

func GetLinks(id uint64) []Link {
	swName := "s" + strconv.FormatUint(id, 10)
	hosts := parseHosts("../mininet/topology.json")
	routes := parseRoutes("routes.json", swName)
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

func GetLinksBytes(id uint64) []LinkBytes {
	linkString := GetLinks(id)
	links := make([]LinkBytes, len(linkString))
	for idx, link := range GetLinks(id) {
		links[idx] = link.toBytes()
	}
	return links
}
