package p4switch

func parseYmlAndAddSuspectFlows(flows []Flow, srcAddr string, dstAddr string){
	var config *SwitchConfig =  ParseSwConfig("s1", "../config/singlesw-config.yml")

    config.Rules = append(config.Rules, Rule{
    	Table:       "MyIngress.ipv4_tag_and_drop",
		Key:         []string{srcAddr,dstAddr},
		Type:        "exact",
		Action:      "",
		ActionParam: []string{},
	})
}
