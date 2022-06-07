package p4switch

import (
	"context"
	"controller/pkg/client"
	"strconv"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
//	"controller/pkg/restapi"
)

type GrpcSwitch struct {
	id         uint64
	conf int
	configName string
	configNameAlt string
	ports      int
	addr       string
	log        *log.Entry
	errCh      chan error
	ctx        context.Context
	cancel     context.CancelFunc
	certFile   string
	p4RtC      *client.Client
	messageCh  chan *p4_v1.StreamMessageResponse
	suspect_flows []Flow
	digests []Digest
}

func (sw *GrpcSwitch) GetName() string {
	return "s" + strconv.FormatUint(sw.id, 10)
}

func (sw *GrpcSwitch) GetLogger() *log.Entry {
	return sw.log
}


func (sw *GrpcSwitch) GetDigests() []Digest{
	return sw.digests
}

func (sw *GrpcSwitch) GetConf() int{
	return sw.conf
}

func (sw *GrpcSwitch) ChangeConf(){
	if sw.conf == 0 {
		sw.conf = 1
	} else{
		sw.conf = 0
	}
}