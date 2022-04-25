package main

import (
	"controller/pkg/util/conversion"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Rule struct {
	table  string
	action string
	ip     string
	mac    string
	port   uint32
}

type RuleBytes struct {
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

type rule struct {
	Port   uint32
	Host   string
	Action string
	Table  string
}

type config struct {
	Rules   []rule
	Program string
	Digest  string
}

func (route *Rule) toBytes() RuleBytes {
	ip, _ := conversion.IpToBinary(route.ip)
	mac, _ := conversion.MacToBinary(route.mac)
	port, _ := conversion.UInt32ToBinaryCompressed(route.port)
	return RuleBytes{
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
		log.Fatalf("Could not parse host file: %s", err)
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

func parseConfig(fileName string, swName string) config {
	// Open our jsonFile
	jsonFile, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Could not parse config file %s: %s", fileName, err)
	}
	defer jsonFile.Close()
	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	var configs map[string]config
	if err = json.Unmarshal(jsonBytes, &configs); err != nil {
		log.Fatal(err)
	}
	return configs[swName]
}

func (sw *GrpcSwitch) GetRules() []Rule {
	links := make([]Rule, 0, 4)
	hosts := parseHosts(p4topology)
	config := parseConfig(sw.configName, sw.GetName())
	// foreach link
	for _, route := range config.Rules {
		// find the host
		for _, host := range hosts {
			if host.name == route.Host {
				links = append(links, Rule{
					table:  route.Table,
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

func GetRulesBytes(routes []Rule) []RuleBytes {
	routeBytes := make([]RuleBytes, len(routes))
	for idx, link := range routes {
		routeBytes[idx] = link.toBytes()
	}
	return routeBytes
}

func (sw *GrpcSwitch) GetProgram() string {
	config := parseConfig(sw.configName, sw.GetName())
	return config.Program
}

func (sw *GrpcSwitch) GetDigest() string {
	config := parseConfig(sw.configName, sw.GetName())
	return config.Digest
}
