#define V1MODEL_VERSION 20200408
#define CPU_PORT 255

#include <core.p4>
#include <v1model.p4>

typedef bit<16> McastGrp_t;

typedef bit<48> MacAddr_t;

header ethernet_t {
    MacAddr_t dstAddr;
    MacAddr_t srcAddr;
    bit<16> etherType;
}

struct metadata {
}

@controller_header("packet_in")
header packet_in_t {
    MacAddr_t dstAddr;
}

@controller_header("packet_out")
header packet_out_t {
    bit<16> egress_spec;
}

struct headers {
    packet_in_t  packetin;
    packet_out_t packetout;
    ethernet_t ethernet;    
}

parser ParserImpl(packet_in packet, out headers hdr, inout metadata meta, inout standard_metadata_t standard_metadata) {

    state start {
	    transition select(standard_metadata.ingress_port){
            CPU_PORT: parse_packet_out;	
	        default: parse_ethernet;
        }
    }
    
    state parse_packet_out {
        packet.extract(hdr.packetout);
        transition parse_ethernet;
    }

    state parse_ethernet {
        packet.extract(hdr.ethernet);
        transition accept;
    }
}

control EgressImpl(inout headers hdr, inout metadata meta, inout standard_metadata_t standard_metadata) {
    apply {
        if (standard_metadata.egress_port == standard_metadata.ingress_port) {
            mark_to_drop(standard_metadata);
        }
    }
}

control IngressImpl(inout headers hdr, inout metadata meta, inout standard_metadata_t standard_metadata) {
    action drop() {
        mark_to_drop(standard_metadata);
    }

    action forward(PortId_t dest_port) {
        standard_metadata.egress_spec = dest_port;
    }

    action learn(){
        standard_metadata.egress_spec = CPU_PORT;
    }

    table dmac {
        key = {
            hdr.ethernet.dstAddr: exact;
        }
        actions = {
            forward;
            learn;
            drop;
            NoAction;
        }
        default_action = learn();
        size = 4096;
    }
    apply {
        dmac.apply();

        // packet out
        if (standard_metadata.ingress_port == CPU_PORT) { 
            standard_metadata.egress_spec = (bit<9>)hdr.packetout.egress_spec;
            hdr.packetout.setInvalid();
        }

        // packet in
        if (standard_metadata.egress_spec == CPU_PORT) { 
            hdr.packetin.setValid();
            hdr.packetin.dstAddr = hdr.ethernet.dstAddr;
        }
    }

    
}

control DeparserImpl(packet_out packet, in headers hdr) {
    apply {
        packet.emit(hdr.packetin);
        packet.emit(hdr.ethernet);
    }
}

control verifyChecksum(inout headers hdr, inout metadata meta) {
    apply { }
}

control computeChecksum(inout headers hdr, inout metadata meta) {
    apply { }
}

V1Switch(p = ParserImpl(),
         ig = IngressImpl(),
         vr = verifyChecksum(),
         eg = EgressImpl(),
         ck = computeChecksum(),
         dep = DeparserImpl()) main;