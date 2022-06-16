package p4switch

import (
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

func (sw *GrpcSwitch) ChangeConfig(configName string) error {
	sw.configName = configName
	if _, err := sw.p4RtC.SaveFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.addRules()
	sw.enableDigest()
	time.Sleep(defaultWait)
	if err := sw.p4RtC.CommitFwdPipe(); err != nil {
		return err
	}
	sw.ChangeConf()
	return nil
}

func (sw *GrpcSwitch) ChangeConfigSync(configName string) error {
	sw.configName = configName
	if _, err := sw.p4RtC.SetFwdPipeFromBytes(sw.readBin(), sw.readP4Info(), 0); err != nil {
		return err
	}
	sw.addRules()
	sw.enableDigest()
	return nil
}

func (sw *GrpcSwitch) addTableEntry(entry *p4_v1.TableEntry) {
	if err := sw.p4RtC.InsertTableEntry(entry); err != nil {
		sw.log.Errorf("Error adding entry: %+v\n%v", entry, err)
		sw.errCh <- err
		return
	}
	sw.log.Tracef("Added entry: %+v", entry)
}

func (sw *GrpcSwitch) removeTableEntry(entry *p4_v1.TableEntry) {
	if err := sw.p4RtC.DeleteTableEntry(entry); err != nil {
		sw.log.Errorf("Error deleting entry: %+v\n%v", entry, err)
		sw.errCh <- err
		return
	}
	sw.log.Tracef("Deleted entry: %+v", entry)
}

func (sw *GrpcSwitch) addRules() {
	entries := getAllTableEntries(sw)
	for _, entry := range entries {
		sw.addTableEntry(entry)
	}
}

func (sw *GrpcSwitch) DropFlow(flow Flow) {

	rule := Rule{
	    	Table:       "MyEgress.ipv4_drop",
			Key:         []string{flow.Attacker.String(),flow.Victim.String()},
			Type:        "exact",
			Action:      "MyEgress.drop",
			ActionParam: []string{},
	}
	tableEntry := createTableEntry(sw, rule)

	sw.addTableEntry(tableEntry)
}

func (sw *GrpcSwitch) RemoveDropFlow(flow Flow) {

	rule := Rule{
	    	Table:       "MyEgress.ipv4_drop",
			Key:         []string{flow.Attacker.String(),flow.Victim.String()},
			Type:        "exact",
			Action:      "MyEgress.drop",
			ActionParam: []string{},
	}
	tableEntry := createTableEntry(sw, rule)

	sw.removeTableEntry(tableEntry)
}

func readFileBytes(filePath string) []byte {
	var bytes []byte
	if filePath != "" {
		var err error
		if bytes, err = ioutil.ReadFile(filePath); err != nil {
			log.Fatalf("Error when reading binary from '%s': %v", filePath, err)
		}
	}
	return bytes
}

func (sw *GrpcSwitch) readP4Info() []byte {
	p4Info := p4Path + sw.getProgramName() + p4InfoExt
	sw.log.Tracef("p4Info %s", p4Info)
	return readFileBytes(p4Info)
}

func (sw *GrpcSwitch) readBin() []byte {
	p4Bin := p4Path + sw.getProgramName() + p4BinExt
	sw.log.Tracef("p4Bin %s", p4Bin)
	return readFileBytes(p4Bin)
}
