package main

import (
	"controller/pkg/client"
	"io/ioutil"
	"time"

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
	sw.log.Debugf("Added %s entry: %d -> p%d", route.table, route.ip, route.port)
}

func (sw *GrpcSwitch) ChangeConfig(configName string) error {
	sw.configName = configName
	if _, err := sw.p4RtC.SaveFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.addRoutes()
	time.Sleep(defaultWait)
	if err := sw.p4RtC.CommitFwdPipe(); err != nil {
		return nil
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
