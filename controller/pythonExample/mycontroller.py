#!/usr/bin/env python3
import argparse
import grpc
import socket
import os
import sys
import codecs
from time import sleep

from scapy.all import get_if_hwaddr, get_if_list, hex_bytes, bind_layers
from scapy.all import Ether, IP, UDP

sys.path.append(".")
from myTunnel_header import MyTunnel

# Import P4Runtime lib from parent utils dir
# Probably there's a better way of doing this.
sys.path.append(
    os.path.join(os.path.dirname(os.path.abspath(__file__)),
                 '../../utils/'))
#print(str(sys.path))
import p4runtime_lib.bmv2
from p4runtime_lib.switch import ShutdownAllSwitchConnections
import p4runtime_lib.helper


TYPE_MYTUNNEL = 0x1212
TYPE_IPV4 = 0x0800


def writeTunnelRules(p4info_helper, sw_id, ing_port, port):
    table_entry = p4info_helper.buildTableEntry(
        table_name="MyIngress.myTunnel_exact",
        match_fields={
            "standard_metadata.ingress_port": ing_port
        },
        action_name="MyIngress.myTunnel_forward",
        action_params={
            "port": port
        })
    sw_id.WriteTableEntry(table_entry)
    print("Installed tunnel rule by forwarding packets with ingress port "+str(ing_port)+" to port "+str(port)+" on switch%s" % sw_id.name)

def writeIpv4Rules(p4info_helper, sw_id, dst_ip_addr, port):
    table_entry = p4info_helper.buildTableEntry(
        table_name="MyIngress.ipv4_lpm",
        match_fields={
            "hdr.ipv4.dstAddr": (dst_ip_addr, 32)
        },
        action_name="MyIngress.ipv4_forward",
        action_params={
          "port": port
        })
    sw_id.WriteTableEntry(table_entry)
    print("Installed ingress forwarding rule on %s" % sw_id.name)

def readTableRules(p4info_helper, sw):
    """
    Reads the table entries from all tables on the switch.

    :param p4info_helper: the P4Info helper
    :param sw: the switch connection
    """
    print('\n----- Reading tables rules for %s -----' % sw.name)
    for response in sw.ReadTableEntries():
        for entity in response.entities:
            entry = entity.table_entry
            # TODO For extra credit, you can use the p4info_helper to translate
            #      the IDs in the entry to names
            table_name = p4info_helper.get_tables_name(entry.table_id)
            print('%s: ' % table_name, end=' ')
            for m in entry.match:
                print(p4info_helper.get_match_field_name(table_name, m.field_id), end=' ')
                print('%r' % (p4info_helper.get_match_field_value(m),), end=' ')
            action = entry.action.action
            action_name = p4info_helper.get_actions_name(action.action_id)
            print('->', action_name, end=' ')
            for p in action.params:
                print(p4info_helper.get_action_param_name(action_name, p.param_id), end=' ')
                print('%r' % p.value, end=' ')
            print()

def printGrpcError(e):
    print("gRPC Error:", e.details(), end=' ')
    status_code = e.code()
    print("(%s)" % status_code.name, end=' ')
    traceback = sys.exc_info()[2]
    print("[%s:%d]" % (traceback.tb_frame.f_code.co_filename, traceback.tb_lineno))


bind_layers(Ether, MyTunnel, type=TYPE_MYTUNNEL)
bind_layers(MyTunnel, IP, pid=TYPE_IPV4)


def main(p4info_file_path, bmv2_file_path, filename):
    # Instantiate a P4Runtime helper from the p4info file
    p4info_helper = p4runtime_lib.helper.P4InfoHelper(p4info_file_path)

    try:
        # Create a switch connection object for s1 and s2;
        # this is backed by a P4Runtime gRPC connection.
        # Also, dump all P4Runtime messages sent to switch to given txt files.

        s1 = p4runtime_lib.bmv2.Bmv2SwitchConnection(
            name='s1',
            address='127.0.0.1:50051',
            device_id=1, 
	        proto_dump_file="p4runtimes1.log.txt")
        
        s2 = p4runtime_lib.bmv2.Bmv2SwitchConnection(
            name='s2',
            address='127.0.0.1:50052',
            device_id=2, 
	        proto_dump_file="p4runtimes2.log.txt")


        # Send master arbitration update message to establish this controller as
        # master (required by P4Runtime before performing any other write operation)
        if (s1.MasterArbitrationUpdate() == None):
            print("Failed to establish the connection")
        if (s2.MasterArbitrationUpdate() == None):
            print("Failed to establish the connection")

        # Install the P4 program on the switches
        s1.SetForwardingPipelineConfig(p4info=p4info_helper.p4info,
                                       bmv2_json_file_path=bmv2_file_path)
        print("Installed P4 Program using SetForwardingPipelineConfig on s1")
        s2.SetForwardingPipelineConfig(p4info=p4info_helper.p4info,
                                      bmv2_json_file_path=bmv2_file_path)
        print("Installed P4 Program using SetForwardingPipelineConfig on s2")
        """
        writeIpv4Rules(p4info_helper, sw_id=s1, dst_ip_addr="10.10.10.1", port = 255)
        writeIpv4Rules(p4info_helper, sw_id=s1, dst_ip_addr="10.10.10.2", port = 255)
        writeIpv4Rules(p4info_helper, sw_id=s1, dst_ip_addr="10.10.3.3",  port = 255)

        writeIpv4Rules(p4info_helper, sw_id=s2, dst_ip_addr="10.10.10.1", port = 255)
        writeIpv4Rules(p4info_helper, sw_id=s2, dst_ip_addr="10.10.10.2", port = 255)
        writeIpv4Rules(p4info_helper, sw_id=s2, dst_ip_addr="10.10.3.3",  port = 255)
        """
        writeTunnelRules(p4info_helper, sw_id=s1, ing_port=255, port=1)
        writeTunnelRules(p4info_helper, sw_id=s2, ing_port=1, port=1)
        writeTunnelRules(p4info_helper, sw_id=s1, ing_port=1, port=255)

        #read all table rules         
        readTableRules(p4info_helper, s1)
        readTableRules(p4info_helper, s2)

        pkt =  Ether(dst='ff:ff:ff:ff:ff:ff')
        pkt = pkt / MyTunnel(dst_id=1) /"ciao"
        packetOutPayload = pkt

        #packetOutPayload = Ether(src="3c:a8:2a:13:8f:bb", dst="00:04:00:00:00:01")/IP(dst="10.10.3.3", src="10.10.10.1")/UDP(sport=7777, dport=80)/"111111112222222233333333"
        #print(packetin)
        #packet = packetin.packet.payload
        packet=bytes(packetOutPayload)
        packetout = p4info_helper.buildPacketOut(
            payload = packet, #send the packet in you received back to output port 3!
            metadata = {1: str.encode("\000\001")} #egress_spec (check @controller_header("packet_out") in the p4 code)
        )
        #Ether(packetOutPayload).show2()
        print("send PACKET OUT")
        Ether(packetout.payload).show2()
        print("METADATA = "+ str(packetout.metadata))
        s1.PacketOut(packetout)
        print("****************************************************************")
        print("****************************************************************")            
        packetin = s1.PacketIn()	    #Packet in!             
        if packetin is not None:
            print("PACKET IN received")
            pktIn=Ether(packetin.packet.payload)
            pktIn.show2()
            #print(pktIn)
            if MyTunnel in pktIn:
                if (pktIn.ts_ing1 > 0) and (pktIn.ts_eg1 > 0) and (pktIn.ts_is2 > 0) and (pktIn.ts_es2 > 0) and (pktIn.ts_ing2 > 0) and (pktIn.ts_eg2 > 0):
                    f = open(filename,"a")
                    f.write(str(pktIn.ts_ing1))
                    f.write(",")
                    f.write(str(pktIn.ts_eg1))
                    f.write(",")
                    f.write(str(pktIn.ts_is2))
                    f.write(",")
                    f.write(str(pktIn.ts_es2))
                    f.write(",")
                    f.write(str(pktIn.ts_ing2))
                    f.write(",")
                    f.write(str(pktIn.ts_eg2))
                    f.write("\n")
                    f.close()
                    sys.stdout.flush()


    except KeyboardInterrupt:
        print(" Shutting down.")
    except grpc.RpcError as e:
        printGrpcError(e)

    ShutdownAllSwitchConnections()

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='P4Runtime Controller')
    parser.add_argument('--p4info', help='p4info proto in text format from p4c',
                        type=str, action="store", required=False,
                        default='./build/simple.p4info.txt')
    parser.add_argument('--bmv2-json', help='BMv2 JSON file from p4c',
                        type=str, action="store", required=False,
                        default='./build/simple.json')
    parser.add_argument('-f', help='The file in which entries are saved - line format is (6 entries): ts_ingress_rs1_1,ts_egress_rs1_1,ts_ingress_rs2,ts_egress_rs2,ts_ingress_rs1_2,ts_egress_rs1_2',
                        type=str, action="store", required=False,
                        default='test.csv')
    args = parser.parse_args()


    filename = args.f

    if not os.path.exists(args.p4info):
        parser.print_help()
        print("\np4info file not found: %s\nHave you run 'make'?" % args.p4info)
        parser.exit(1)
    if not os.path.exists(args.bmv2_json):
        parser.print_help()
        print("\nBMv2 JSON file not found: %s\nHave you run 'make'?" % args.bmv2_json)
        parser.exit(1)
    
    main(args.p4info, args.bmv2_json, filename)


