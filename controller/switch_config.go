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
	p4InfoExt = ".p4info.txt"
	p4BinExt  = ".json"
	p4Path    = "../p4/"
)

func (sw *GrpcSwitch) addRoutes() {
	for _, route := range sw.GetRules() {
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

func (sw *GrpcSwitch) addIpv4Lpm(route RuleBytes) {
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
		sw.log.Errorf("Error adding %s entry: %d -> p%d", strings.Split(route.table, ".")[1], route.ip, route.port)
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
	sw.enableDigest()
	time.Sleep(defaultWait)
	if err := sw.p4RtC.CommitFwdPipe(); err != nil {
		return err
	}
	return nil
}

func (sw *GrpcSwitch) readP4Info() []byte {
	p4Info := p4Path + sw.GetProgram() + p4InfoExt
	sw.log.Tracef("p4Info %s", p4Info)
	return readFileBytes(p4Info)
}

func (sw *GrpcSwitch) readBin() []byte {
	p4Bin := p4Path + sw.GetProgram() + p4BinExt
	sw.log.Tracef("p4Bin %s", p4Bin)
	return readFileBytes(p4Bin)
}

func (sw *GrpcSwitch) enableDigest() error {
	digestName := sw.GetDigest()
	if digestName == "" {
		sw.log.Debug("Digest not enabled")
		return nil
	}
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
