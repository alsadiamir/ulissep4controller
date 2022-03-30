package main

import (
	"controller/pkg/client"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
)

const (
	p4InfoExt  = ".p4info.txt"
	p4BinExt   = ".json"
	configExt  = "_config.json"
	configPath = "../p4/"
)

func (sw *GrpcSwitch) addRoutes() {
	config := configPath + sw.configName + configExt
	routes := GetRoutes(sw.id, config)
	for _, route := range routes {
		sw.addIpv4Lpm(route.toBytes())
	}
}

func readFileBytes(filePath string) []byte {
	bytes := []byte("per")
	if filePath != "" {
		var err error
		if bytes, err = ioutil.ReadFile(filePath); err != nil {
			log.Fatalf("Error when reading binary from '%s': %v", filePath, err)
		}
	}
	return bytes
}

func (sw *GrpcSwitch) addIpv4Lpm(route RouteBytes) {
	entry := sw.p4RtC.NewTableEntry(
		route.table,
		[]client.MatchInterface{&client.LpmMatch{
			Value: route.ip,
			PLen:  32,
		}},
		sw.p4RtC.NewTableActionDirect(route.action, [][]byte{route.mac, route.port}),
		nil,
	)
	if err := sw.p4RtC.InsertTableEntry(entry); err != nil {
		sw.errCh <- err
		return
	}
	sw.log.Debugf("Added %s entry: %d -> p%d", strings.Split(route.table, ".")[1], route.ip, route.port)
}

func (sw *GrpcSwitch) ChangeConfig(configName string) error {
	sw.configName = configName
	if _, err := sw.p4RtC.SaveFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.addRoutes()
	sw.enableDigest(digestName)
	time.Sleep(defaultWait)
	if err := sw.p4RtC.CommitFwdPipe(); err != nil {
		return err
	}
	return nil
}

func (sw *GrpcSwitch) readP4Info() []byte {
	p4Info := configPath + sw.configName + p4InfoExt
	return readFileBytes(p4Info)
}

func (sw *GrpcSwitch) readBin() []byte {
	p4Bin := configPath + sw.configName + p4BinExt
	return readFileBytes(p4Bin)
}

func (sw *GrpcSwitch) enableDigest(digestName string) error {
	digestConfig := &p4_v1.DigestEntry_Config{
		MaxTimeoutNs: 0,
		MaxListSize:  1,
		AckTimeoutNs: time.Second.Nanoseconds() * 1000,
	}
	if err := sw.p4RtC.EnableDigest(digestName, digestConfig); err != nil {
		return fmt.Errorf("cannot enable digest %s", digestName)
	}
	sw.log.Debugf("Enabled digest %s", digestName)
	return nil
}
