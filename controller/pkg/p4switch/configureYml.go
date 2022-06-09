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

/*

    var config *p4switch.SwitchConfig =  p4switch.ParseSwConfig("s1", "../config/singlesw-config.yml")

    config.Rules = append(config.Rules, p4switch.Rule{
    	Table:       "MyIngress.ipv4_lpm",
		Key:         "10.0.1.1/24",
		Type:        "",
		Action:      "MyIngress.ipv4_forward",
		ActionParam: []string{"08:00:00:00:01:00","1"},
	})



    fmt.Println(" --- YAML with maps and arrays ---")
    fmt.Println("+%v",config)

*/