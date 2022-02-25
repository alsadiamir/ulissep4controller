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
#define BYTE_COUNT_WIDTH 48
//#define PACKET_BYTE_COUNT_WIDTH (PACKET_COUNT_WIDTH + BYTE_COUNT_WIDTH)
#define PACKET_BYTE_COUNT_WIDTH 80

#define PACKET_COUNT_RANGE (PACKET_BYTE_COUNT_WIDTH-1):BYTE_COUNT_WIDTH
#define BYTE_COUNT_RANGE (BYTE_COUNT_WIDTH-1):0

#define FLOW_TABLE_SIZE_EACH 1024
#define HASH_BASE 10w0
#define HASH_MAX 10w1023
typedef bit<PACKET_BYTE_COUNT_WIDTH> PacketByteCountState_t;


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
    bit<32> flow;
    bit<32> flow_opp;
    bit<16> treshold;
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
    
    //direct_counter(CounterType.packets) c;
    register<bit<48>>(1024) last_seen;
    register<bit<48>>(1024) window;
    register<bit<16>>(1024) treshold;

    action drop() {
        mark_to_drop(standard_metadata);
    }

    action update_packet_gap(bit<32> flow_id) {
      bit<48> last_pkt_cnt;
      /* Get the time the previous packet was seen */
      last_seen.read(last_pkt_cnt,flow_id);
      /* Update the register with the new timestamp */
      last_seen.write((bit<32>)flow_id, last_pkt_cnt + 1);
    }

    action reset_flow(bit<32> flow,bit<32> flow_opp) {
      last_seen.write((bit<32>)flow,0);
      last_seen.write((bit<32>)flow_opp,0);
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
        //counters = c;
        actions = {
            ipv4_forward;
            drop;
            NoAction;
        }
        size = 1024;
        default_action = NoAction();
    }
    
    apply {
        if (hdr.ipv4.isValid()) {    
            bit<32> flow;
            bit<32> flow_opp;      
            bit<48> last_pkt_cnt;
            bit<48> last_pkt_cnt_opp;
	        bit<48> last_time;
	        bit<48> intertime;
            bit<16> current_treshold;
            bit<48> diff_pkt_cnt;

            hash(flow, HashAlgorithm.crc16, HASH_BASE,
                {hdr.ipv4.srcAddr, 7w11, hdr.ipv4.dstAddr}, HASH_MAX);
            hash(flow_opp, HashAlgorithm.crc16, HASH_BASE,
                {hdr.ipv4.dstAddr, 7w11, hdr.ipv4.srcAddr}, HASH_MAX);
            digest<digest_t>(0, {flow, flow_opp, (bit<16>)0});      

            // read flow values
            window.read(last_time,flow);
            treshold.read(current_treshold,flow);


            
            // first time initialize 
            if (last_time == (bit<48>)0) {
                window.write(flow,standard_metadata.ingress_global_timestamp);
                last_time = standard_metadata.ingress_global_timestamp;
            }
            // increase current treshold
            if (current_treshold == (bit<16>)0) {
                treshold.write(flow,(bit<16>)200);
                current_treshold = 200;
            }
            
            // update flow timestamp
            window.write(flow,standard_metadata.ingress_global_timestamp);

            // check window
            intertime = standard_metadata.ingress_global_timestamp - last_time;
            if (intertime > WINDOW_SIZE) {
                reset_flow(flow,flow_opp);
            }
            last_seen.read(last_pkt_cnt,flow);
            last_seen.read(last_pkt_cnt_opp,flow_opp);
            diff_pkt_cnt = last_pkt_cnt - last_pkt_cnt_opp + 1;

            if (diff_pkt_cnt < (bit<48>)current_treshold) {
                update_packet_gap(flow);
            } else {
                // treshold is reached if you reach the max tresh
                if(current_treshold > MAX_TRESHOLD){
                    drop();
                } else {
                    //raise your treshold, and reset the flow 
                    treshold.write(flow,current_treshold+200);
                    reset_flow(flow,flow_opp);
                }
            }
            ipv4_lpm.apply(); 
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