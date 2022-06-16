/* -*- P4_16 -*- */
#include <core.p4>
#include <v1model.p4>


#define PACKETS 131072
#define PACKET_COUNTER_WIDTH 17

//up to 1 packet over 16
#define SAMPLING_COUNTER_WIDTH 4

#define WINDOW 30000000


typedef bit<9>  egressSpec_t;
typedef bit<48> macAddr_t;
typedef bit<32> ip4Addr_t;



#include "includes/headers.p4"

#include "includes/registers.p4"

#include "includes/parser.p4"



control MyVerifyChecksum(inout headers hdr, inout metadata meta) {   
    apply {  }
}


control MyIngress(inout headers hdr,
                  inout metadata meta,
                  inout standard_metadata_t standard_metadata) {

    action drop() {
        mark_to_drop(standard_metadata);
        exit;
    }

    action send_digest() {
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

            if(ipv4_tag_and_drop.apply().hit){
                send_digest();
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
