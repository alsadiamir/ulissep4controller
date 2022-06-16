/* -*- P4_16 -*- */
#include <core.p4>
#include <v1model.p4>

const bit<16> TYPE_IPV4 = 0x800;
#define TRESHOLD 2500
//microseconds
#define WINDOW_SIZE 30000000

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
    bit<48> min_flow;
    bit<48> min_flow_opp;
}

struct headers {
    ethernet_t   ethernet;
    ipv4_t       ipv4;
}

struct digest_t {
    bit<16> type; //=0, asymmetric
    ip4Addr_t srcAddr;
    ip4Addr_t dstAddr;
    bit<9>    srcPort;
    bit<9>    dstPort;
    bit<32>   flow;
    bit<8>  swap;
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

    register<bit<48>>(1024) pkt_count_0;
    register<bit<48>>(1024) pkt_count_1;
    register<bit<48>>(1024) flow_count_treshold;
    register<bit<48>>(1024) last_seen;


    action drop() {
        mark_to_drop(standard_metadata);
        exit;
    }

    action ipv4_forward(macAddr_t dstAddr, egressSpec_t port) {
        standard_metadata.egress_spec = port;
        hdr.ethernet.srcAddr = hdr.ethernet.dstAddr;
        hdr.ethernet.dstAddr = dstAddr;
        hdr.ipv4.ttl = hdr.ipv4.ttl - 1;
    }

    action send_digest(bit<32> flow, bit<8> swap) {
        digest<digest_t>(0, {0,hdr.ipv4.srcAddr, hdr.ipv4.dstAddr, standard_metadata.ingress_port ,standard_metadata.egress_spec, flow, swap});
    }

    action find_min(bit<48> pkt_cnt0, bit<48> pkt_cnt1, bit<48> pkt_cnt_opp0, bit<48> pkt_cnt_opp1){

        //MIN of FLOW
        if(pkt_cnt0 > pkt_cnt1){
            meta.min_flow = pkt_cnt1;
        } else{
            meta.min_flow = pkt_cnt0;
        }

        //MIN of FLOW OPP
        if(pkt_cnt_opp0 > pkt_cnt_opp1){
            meta.min_flow_opp = pkt_cnt_opp1;
        } else{
            meta.min_flow_opp = pkt_cnt_opp0;
        }  
    }

    table ipv4_lpm {
        key = {
            hdr.ipv4.dstAddr: lpm;
        }
        actions = {
            ipv4_forward;
            //drop;
            NoAction;
        }
        size = 1024;
        support_timeout = true;
        default_action = NoAction();
    }

    apply {
        if (hdr.ipv4.isValid()) {
            ipv4_lpm.apply();

            bit<48> flow_hit;
            bit<32> flow0;
            bit<32> flow_opp0;
            bit<48> last_pkt_cnt0;
            bit<48> last_pkt_cnt_opp0;

            bit<32> flow1;
            bit<32> flow_opp1;
            bit<48> last_pkt_cnt1;
            bit<48> last_pkt_cnt_opp1;

            bit<48> last_time;
            bit<48> diff_time;
            bit<48> diff_pkt_cnt;

            // compute flow index 0
            hash(flow0,     HashAlgorithm.crc32, HASH_BASE, {hdr.ipv4.srcAddr, 7w11, hdr.ipv4.dstAddr}, HASH_MAX);
            hash(flow_opp0, HashAlgorithm.crc32, HASH_BASE, {hdr.ipv4.dstAddr, 7w11, hdr.ipv4.srcAddr}, HASH_MAX);

            // compute flow index 1
            hash(flow1,     HashAlgorithm.crc16, HASH_BASE, {hdr.ipv4.srcAddr, 7w11, hdr.ipv4.dstAddr}, HASH_MAX);
            hash(flow_opp1, HashAlgorithm.crc16, HASH_BASE, {hdr.ipv4.dstAddr, 7w11, hdr.ipv4.srcAddr}, HASH_MAX);

            //read packet count index 0
            pkt_count_0.read(last_pkt_cnt0,     flow0);
            pkt_count_0.read(last_pkt_cnt_opp0, flow_opp0);

            //read packet count index 1
            pkt_count_1.read(last_pkt_cnt1,     flow1);
            pkt_count_1.read(last_pkt_cnt_opp1, flow_opp1);

            //updating the packet count index 0
            last_pkt_cnt0 = last_pkt_cnt0 + 1;
            pkt_count_0.write(flow0, last_pkt_cnt0);

            //updating the packet count index 1
            last_pkt_cnt1 = last_pkt_cnt1 + 1;
            pkt_count_1.write(flow1, last_pkt_cnt1);

            //finding the minimum of FLOW and FLOW OPP - using last_pkt_cnt0 and last_pkt_cnt_opp0 to keep the minimum
            find_min(last_pkt_cnt0,last_pkt_cnt1,last_pkt_cnt_opp0,last_pkt_cnt_opp1);
            last_pkt_cnt0 = meta.min_flow;
            last_pkt_cnt_opp0 = meta.min_flow_opp;

            //calculating the difference
            if (last_pkt_cnt0 > last_pkt_cnt_opp0) {
                diff_pkt_cnt = last_pkt_cnt0 - last_pkt_cnt_opp0 + 1;
            } else {
                diff_pkt_cnt = last_pkt_cnt_opp0 - last_pkt_cnt0 + 1;
            }

            //sending the digest if flow is hit
            if (diff_pkt_cnt > (bit<48>)TRESHOLD) {
                flow_count_treshold.read(flow_hit,     flow0);
                if(flow_hit == (bit<48>)0) {
                    flow_count_treshold.write(flow0, (bit<48>)1);
                    send_digest((bit<32>)flow0,0);
                }
            }

            // read the previous timestamp when the flow was hit
            last_seen.read(last_time,flow0);
            // check window
            diff_time = standard_metadata.ingress_global_timestamp - last_time;

            //checking if the window is expired
            if (diff_time > (bit<48>)WINDOW_SIZE) {
                
                //resetting the flow counters
                pkt_count_0.write(flow0,(bit<48>)0);
                pkt_count_0.write(flow_opp0,(bit<48>)0);
                pkt_count_1.write(flow1,(bit<48>)0);
                pkt_count_1.write(flow_opp1,(bit<48>)0);
                
                //only updating the first flow
                last_seen.write(flow0,standard_metadata.ingress_global_timestamp);

                //sending swap signal
                send_digest(flow0,1);
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
