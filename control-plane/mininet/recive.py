#!/usr/bin/env python3

from scapy.all import sniff, get_if_list
import sys


def get_if():
    iface = None  # "h1-eth0"
    for i in get_if_list():
        if "eth0" in i:
            iface = i
            break
    if not iface:
        print("Cannot find eth0 interface")
        exit(1)
    return iface


def handle_pkt(pkt):
    print("new packet")
    # pkt.show2()
    sys.stdout.flush()


def main():
    iface = 'eth0'
    print("sniffing on %s" % iface)
    sys.stdout.flush()
    sniff(filter="udp and port 4321", iface=iface, prn=lambda x: handle_pkt(x))


if __name__ == '__main__':
    main()
