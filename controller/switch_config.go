package main

import (
	"fmt"
	"io/ioutil"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
)

const (
	p4InfoExt = ".p4info.txt"
	p4BinExt  = ".json"
	p4Path    = "../p4/"
)

var digestConfig p4_v1.DigestEntry_Config = p4_v1.DigestEntry_Config{
	MaxTimeoutNs: 0,
	MaxListSize:  1,
	AckTimeoutNs: time.Second.Nanoseconds() * 1000,
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

func (sw *GrpcSwitch) addTableEntry(entry *p4_v1.TableEntry) {
	if err := sw.p4RtC.InsertTableEntry(entry); err != nil {
		sw.log.Errorf("Error adding entry: %+v\n%v", entry, err)
		sw.errCh <- err
		return
	}
	sw.log.Tracef("Added entry: %+v", entry)
}

func (sw *GrpcSwitch) addRoutes() {
	entries := sw.getAllTableEntries()
	for _, entry := range entries {
		sw.addTableEntry(entry)
	}
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

func (sw *GrpcSwitch) ChangeConfigSync(configName string) error {
	sw.configName = configName
	if _, err := sw.p4RtC.SetFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.addRoutes()
	sw.enableDigest()
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
	digestName := sw.GetDigests()
	for _, digest := range digestName {
		if digest == "" {
			continue
		}
		if err := sw.p4RtC.EnableDigest(digest, &digestConfig); err != nil {
			return fmt.Errorf("cannot enable digest %s", digest)
		}
		sw.log.Debugf("Enabled digest %s", digest)
	}
	return nil
}
