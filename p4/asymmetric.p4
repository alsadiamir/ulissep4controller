/* -*- P4_16 -*- */
#include <core.p4>
#include <v1model.p4>

const bit<16> TYPE_IPV4 = 0x800;
typedef bit<10> PortId_t;
const PortId_t NUM_PORTS = 512;
#define PACKET_COUNT_WIDTH 32
#define TRESHOLD 200
#define MAX_TRESHOLD 1000
//microseconds
#define WINDOW_SIZE 15000000

#define HASH_BASE 10w0
#define HASH_MAX 10w1023

/*************************************************************************
*********************** H E A D E R S  ***********************************
*************************************************************************/

typedef bit<9>  egressSpec_t;
typedef bit<48> macAddr_t;
typedef bit<32> ip4Addr_t;

header ethernet_t {
    macAddr_t dstAddr;
    macAddr_t srcAddr;
    bit<16>   etherType;
}

header ipv4_t {
    bit<4>    version;
    bit<4>    ihl;
    bit<8>    diffserv;
    bit<16>   totalLen;
    bit<16>   identification;
    bit<3>    flags;
    bit<13>   fragOffset;
    bit<8>    ttl;
    bit<8>    protocol;
    bit<16>   hdrChecksum;
    ip4Addr_t srcAddr;
    ip4Addr_t dstAddr;
    ip4Addr_t realsrcAddr;
    ip4Addr_t realdstAddr;
}

struct metadata {
}

struct headers {
    ethernet_t   ethernet;
    ipv4_t       ipv4;
}

struct digest_t {
    ip4Addr_t srcAddr;
    ip4Addr_t dstAddr;
    bit<9>    srcPort;
    bit<9>    dstPort;
}

/*************************************************************************
*********************** P A R S E R  ***********************************
*************************************************************************/

parser MyParser(packet_in packet,
                out headers hdr,
                inout metadata meta,
                inout standard_metadata_t standard_metadata) {

    state start {
        transition parse_ethernet;
    }

    state parse_ethernet {
        packet.extract(hdr.ethernet);
        transition select(hdr.ethernet.etherType) {
            TYPE_IPV4: parse_ipv4;
            default: accept;
        }
    }

    state parse_ipv4 {
        packet.extract(hdr.ipv4);
        transition accept;
    }

}

/*************************************************************************
************   C H E C K S U M    V E R I F I C A T I O N   *************
*************************************************************************/

control MyVerifyChecksum(inout headers hdr, inout metadata meta) {
    apply {  }
}


/*************************************************************************
**************  I N G R E S S   P R O C E S S I N G   *******************
*************************************************************************/

control MyIngress(inout headers hdr,
                  inout metadata meta,
                  inout standard_metadata_t standard_metadata) {

    register<bit<48>>(1024) pkt_count;
    register<bit<48>>(1024) last_seen;

    action drop() {
        mark_to_drop(standard_metadata);
        exit;
    }

    action update_pkt_count(bit<32> flow_id) {
      bit<48> last_pkt_cnt;
      /* Get the time the previous packet was seen */
      pkt_count.read(last_pkt_cnt, flow_id);
      /* Update the register with the new timestamp */
      pkt_count.write((bit<32>)flow_id, last_pkt_cnt + 1);
    }

    action reset_pkt_count(bit<32> flow, bit<32> flow_opp) {
      pkt_count.write((bit<32>)flow,0);
      pkt_count.write((bit<32>)flow_opp,0);
    }

    action ipv4_forward(macAddr_t dstAddr, egressSpec_t port) {
        standard_metadata.egress_spec = port;
        hdr.ethernet.srcAddr = hdr.ethernet.dstAddr;
        hdr.ethernet.dstAddr = dstAddr;
        hdr.ipv4.ttl = hdr.ipv4.ttl - 1;
    }

    action send_digest() {
        //digest<digest_t>(0, {hdr.ipv4.srcAddr, hdr.ipv4.dstAddr, standard_metadata.ingress_port ,standard_metadata.egress_spec});
        // inverted idk why
        digest<digest_t>(0, {hdr.ipv4.dstAddr, hdr.ipv4.srcAddr, standard_metadata.egress_spec ,standard_metadata.ingress_port});
    }

    table ipv4_lpm {
        key = {
            hdr.ipv4.dstAddr: lpm;
        }
        actions = {
            ipv4_forward;
            drop;
            NoAction;
        }
        size = 1024;
        support_timeout = true;
        default_action = drop();
    }

    table ipv4_drop {
        key = {
            hdr.ipv4.srcAddr: exact;
        }
        actions = {
            NoAction;
            drop;
        }
        size = 1024;
        support_timeout = true;
        default_action = NoAction();
    }

    apply {
        if (hdr.ipv4.isValid()) {
            ipv4_lpm.apply();
            ipv4_drop.apply();

            bit<32> flow;
            bit<32> flow_opp;
            bit<48> last_pkt_cnt;
            bit<48> last_pkt_cnt_opp;
	        bit<48> last_time;
	        bit<48> intertime;
            bit<48> diff_pkt_cnt;

            // compute flow index
            hash(flow,     HashAlgorithm.crc16, HASH_BASE, {hdr.ipv4.srcAddr, 7w11, hdr.ipv4.dstAddr}, HASH_MAX);
            hash(flow_opp, HashAlgorithm.crc16, HASH_BASE, {hdr.ipv4.dstAddr, 7w11, hdr.ipv4.srcAddr}, HASH_MAX);

            // read flow values
            last_seen.read(last_time,flow);
            last_seen.write(flow,standard_metadata.ingress_global_timestamp);

            // first time initialize
            if (last_time == (bit<48>)0) {
                last_seen.write(flow,standard_metadata.ingress_global_timestamp);
                last_time = standard_metadata.ingress_global_timestamp;
            }

            // check window
            intertime = standard_metadata.ingress_global_timestamp - last_time;
            if (intertime > WINDOW_SIZE) {
                reset_pkt_count(flow,flow_opp);
                return;
            }
            pkt_count.read(last_pkt_cnt,     flow);
            pkt_count.read(last_pkt_cnt_opp, flow_opp);
            diff_pkt_cnt = last_pkt_cnt - last_pkt_cnt_opp + 1;

            if (diff_pkt_cnt < (bit<48>)TRESHOLD) {
                update_pkt_count(flow);
            } else {
                send_digest();
            }
        }
    }
}

/*************************************************************************
****************  E G R E S S   P R O C E S S I N G   *******************
*************************************************************************/

control MyEgress(inout headers hdr,
                 inout metadata meta,
                 inout standard_metadata_t standard_metadata) {
    apply {  }
}

/*************************************************************************
*************   C H E C K S U M    C O M P U T A T I O N   **************
*************************************************************************/

control MyComputeChecksum(inout headers  hdr, inout metadata meta) {
     apply {
	update_checksum(
	    hdr.ipv4.isValid(),
            { hdr.ipv4.version,
	      	  hdr.ipv4.ihl,
              hdr.ipv4.diffserv,
              hdr.ipv4.totalLen,
              hdr.ipv4.identification,
              hdr.ipv4.flags,
              hdr.ipv4.fragOffset,
              hdr.ipv4.ttl,
              hdr.ipv4.protocol,
              hdr.ipv4.srcAddr,
              hdr.ipv4.dstAddr },
            hdr.ipv4.hdrChecksum,
            HashAlgorithm.csum16);
    }
}

/*************************************************************************
***********************  D E P A R S E R  *******************************
*************************************************************************/

control MyDeparser(packet_out packet, in headers hdr) {
    apply {
        packet.emit(hdr.ethernet);
        packet.emit(hdr.ipv4);
    }
}

/*************************************************************************
***********************  S W I T C H  *******************************
*************************************************************************/

V1Switch(
MyParser(),
MyVerifyChecksum(),
MyIngress(),
MyEgress(),
MyComputeChecksum(),
MyDeparser()
) main;