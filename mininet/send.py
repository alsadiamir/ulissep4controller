#!/usr/bin/env python3

from scapy.all import sendp, sniff, get_if_list, get_if_hwaddr
from scapy.packet import Packet
from scapy.layers.l2 import Ether
from scapy.layers.inet import IP, ICMP, UDP
import sys
import socket
from time import sleep


def get_if():
    ifs = get_if_list()
    iface = None  # "h1-eth0"
    for i in get_if_list():
        if "eth0" in i:
            iface = i
            break
    if not iface:
        print("Cannot find eth0 interface")
        exit(1)
    return iface


def main():
    print("main")
    addr = socket.gethostbyname("10.0.1.2")
    iface = get_if()

    pkt = Ether(src=get_if_hwaddr(iface), dst="ff:ff:ff:ff:ff:ff")
    pkt = pkt / IP(dst=addr) / UDP(dport=4321, sport=1234) / "ciao"
    pkt.show2()
    sendp(pkt, iface=iface)
    try:
        for i in range(int(100)):
            sendp(pkt, iface=iface)
            sleep(5)
    except KeyboardInterrupt:
        raise


if __name__ == '__main__':
    main()
