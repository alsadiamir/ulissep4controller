/* -*- P4_16 -*- */
#include <core.p4>
#include <v1model.p4>


#define PACKETS 131072
#define PACKET_COUNTER_WIDTH 17

//up to 1 packet over 16
#define SAMPLING_COUNTER_WIDTH 4

#define TRESHOLD 2500
//microseconds
#define WINDOW_SIZE 30000000

#define HASH_BASE 10w0
#define HASH_MAX 10w1023


typedef bit<9>  egressSpec_t;
typedef bit<48> macAddr_t;
typedef bit<32> ip4Addr_t;



#include "includes/headers.p4"

#include "includes/registers.p4"

#include "includes/parser.p4"

/*
struct digestA_t {
    bit<16> type; //=0, asymmetric
    ip4Addr_t srcAddr;
    ip4Addr_t dstAddr;
    bit<9>    srcPort;
    bit<9>    dstPort;
    bit<32>   flow;
    bit<8>  swap;
}
*/

control MyVerifyChecksum(inout headers hdr, inout metadata meta) {   
    apply {  }
}


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

    action send_digest_lucid() {
        digest<digest_t>(0, {
            1,
            meta.ingress_timestamp,
            meta.packet_length, 
            meta.ip_flags,
            meta.tcp_len,
            meta.tcp_ack,
            meta.tcp_flags,
            meta.tcp_window_size,
            meta.udp_len,
            meta.icmp_type,
            meta.srcPort,
            meta.dstPort,
            meta.src_ip,
            meta.dst_ip,
            meta.ip_upper_protocol,
            meta.swap});
    }

    action send_digest_asym(bit<32> flow, bit<8> swap) {
        digest<digest_t>(0, {
            1,
            0,
            0, 
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            meta.srcPort,
            meta.dstPort,
            meta.src_ip,
            meta.dst_ip,
            0,
            meta.swap});
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

    action ipv4_forward(macAddr_t dstAddr, egressSpec_t port) {
        standard_metadata.egress_spec = port;
        hdr.ethernet.srcAddr = hdr.ethernet.dstAddr;
        hdr.ethernet.dstAddr = dstAddr;
        hdr.ipv4.ttl = hdr.ipv4.ttl - 1;
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

    table ipv4_tag_and_drop {
        key = {
            hdr.ipv4.srcAddr: exact;
            hdr.ipv4.dstAddr: exact;
        }
        actions = {
            //drop;
            NoAction;
        }
        size = 1024;
        support_timeout = true;
        default_action = NoAction();
    }    

    table ipv4_drop {
        key = {
            hdr.ipv4.srcAddr: exact;
            hdr.ipv4.dstAddr: exact;
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
                    send_digest_asym((bit<32>)flow0,0);
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
            }

            if(ipv4_tag_and_drop.apply().hit){
                send_digest_lucid();
            }   

            ipv4_drop.apply();
            
            meta.swap = (bit<16>) 0;
            
        }// valid ipv4_address
    }// apply
}// MyIngress

/*************************************************************************
****************  E G R E S S   P R O C E S S I N G   *******************
*************************************************************************/

control MyEgress(inout headers hdr,
                 inout metadata meta,
                 inout standard_metadata_t standard_metadata) {
    action drop() {
        mark_to_drop(standard_metadata);
        exit;
    }

    table ipv4_drop {
        key = {
            hdr.ipv4.srcAddr: exact;
            hdr.ipv4.dstAddr: exact;
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
            ipv4_drop.apply();                  
        }// valid ipv4_address

    }
}//MyEgress

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
        packet.emit(hdr.icmp);
        packet.emit(hdr.udp);
        packet.emit(hdr.tcp);
        
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
