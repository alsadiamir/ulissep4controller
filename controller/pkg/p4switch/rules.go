package p4switch

import (
	"controller/pkg/client"
	"controller/pkg/util/conversion"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

const macRegexp = "([0-9a-fA-F]{2}[:-]){5}([0-9a-fA-F]{2})"

type Rule struct {
	Table       string
	Key         string
	Type        string
	Action      string
	ActionParam []string `yaml:"action_param"`
}

type SwitchConfig struct {
	Rules   []Rule
	Program string
	Digest  []string
}

func parseSwConfig(swName string, configFileName string) (*SwitchConfig, error) {
	configs := make(map[string]SwitchConfig)
	configFile, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(configFile, &configs); err != nil {
		return nil, err
	}
	config := configs[swName]
	if config.Program == "" {
		return nil, fmt.Errorf("switch config not found in file %s", configFileName)
	}
	return &config, nil
}

func (sw *GrpcSwitch) getProgramName() string {
	config, err := parseSwConfig(sw.GetName(), sw.configName)
	if err != nil {
		sw.log.Errorf("Error getting program name: %v", err)
		return ""
	}
	return config.Program
}

func (sw *GrpcSwitch) getDigests() []string {
	config, err := parseSwConfig(sw.GetName(), sw.configName)
	if err != nil {
		sw.log.Errorf("Error getting digest list: %v", err)
		return make([]string, 0)
	}
	return config.Digest
}

func getAllTableEntries(sw *GrpcSwitch) []*p4_v1.TableEntry {
	var tableEntries []*p4_v1.TableEntry
	config, err := parseSwConfig(sw.GetName(), sw.configName)
	if err != nil {
		sw.log.Errorf("Error getting table entries: %v", err)
		return tableEntries
	}
	for _, rule := range config.Rules {
		tableEntries = append(tableEntries, createTableEntry(sw, rule))
	}
	return tableEntries
}

func createTableEntry(sw *GrpcSwitch, rule Rule) *p4_v1.TableEntry {
	return sw.p4RtC.NewTableEntry(
		rule.Table,
		parseMatchInterface(rule.Type, rule.Key),
		sw.p4RtC.NewTableActionDirect(rule.Action, parseActionParams(rule.ActionParam)),
		nil,
	)
}

func parseActionParams(actionParams []string) [][]byte {
	actionByte := make([][]byte, len(actionParams))
	r, _ := regexp.Compile(macRegexp)
	for i, action := range actionParams {
		// check if it is a mac address
		if r.MatchString(action) {
			actionByte[i], _ = conversion.MacToBinary(action)
		} else {
			num, _ := strconv.ParseInt(action, 10, 64)
			actionByte[i], _ = conversion.UInt64ToBinaryCompressed(uint64(num))
		}
	}
	return actionByte
}

func parseMatchInterface(matchType string, key string) []client.MatchInterface {
	//var matchInterface p4_v1.FieldMatch
	switch matchType {
	case "exact":
		ip, err := conversion.IpToBinary(key)
		if err != nil {
			log.Errorf("Error parsing ip %s", ip)
		}
		return []client.MatchInterface{&client.ExactMatch{
			Value: ip,
		}}
	default:
		values := strings.Split(key, "/")
		if len(values) != 2 {
			log.Errorf("Error parsing match %s -> %s", matchType, key)
			return nil
		}
		ip, err := conversion.IpToBinary(values[0])
		if err != nil {
			log.Errorf("Error parsing ip %v", ip)
		}
		lpm, err := strconv.ParseInt(values[1], 10, 64)
		if err != nil {
			log.Errorf("Error parsing lpm %d", lpm)
		}
		return []client.MatchInterface{&client.LpmMatch{
			Value: ip,
			PLen:  int32(lpm),
		}}
	}
}
