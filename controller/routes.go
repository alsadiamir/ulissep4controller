package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/antoninbas/p4runtime-go-client/pkg/util/conversion"
	log "github.com/sirupsen/logrus"
)

type Route struct {
	table  string
	action string
	ip     string
	mac    string
	port   uint32
}

type RouteBytes struct {
	table  string
	action string
	ip     []byte
	mac    []byte
	port   []byte
}

type host struct {
	name string
	Ip   string
	Mac  string
}

func (route *Route) toBytes() RouteBytes {
	ip, _ := conversion.IpToBinary(route.ip)
	mac, _ := conversion.MacToBinary(route.mac)
	port, _ := conversion.UInt32ToBinaryCompressed(route.port)
	return RouteBytes{
		table:  route.table,
		action: route.action,
		ip:     ip,
		mac:    mac,
		port:   port,
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

type route struct {
	Port   uint32
	Host   string
	Action string
}

type config struct {
	Routes []route
	Table  string
}

func parseConfig(fileName string, swName string) config {
	// Open our jsonFile
	jsonFile, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()
	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	var configs map[string]config
	if err = json.Unmarshal(jsonBytes, &configs); err != nil {
		log.Fatal(err)
	}
	return configs[swName]
}

func GetRoutes(id uint64, configFile string) []Route {
	links := make([]Route, 0, 4)
	swName := "s" + strconv.FormatUint(id, 10)
	hosts := parseHosts("../mininet/topology.json")
	config := parseConfig(configFile, swName)
	// foreach link
	for _, route := range config.Routes {
		// find the host
		for _, host := range hosts {
			if host.name == route.Host {
				links = append(links, Route{
					table:  config.Table,
					action: route.Action,
					ip:     strings.Split(host.Ip, "/")[0],
					mac:    host.Mac,
					port:   route.Port,
				})
			}
		}
	}
	return links
}

func GetRoutesBytes(routes []Route) []RouteBytes {
	routeBytes := make([]RouteBytes, len(routes))
	for idx, link := range routes {
		routeBytes[idx] = link.toBytes()
	}
	return routeBytes
}
