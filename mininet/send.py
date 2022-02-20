#!/usr/bin/env python3

from scapy.all import sendp,  get_if_hwaddr, get_if_list
from scapy.layers.l2 import Ether
from scapy.layers.inet import IP, UDP
import sys
import socket
from time import sleep, time


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


def main():

    if len(sys.argv) < 1:
        print('pass 1 arguments: <destination> ')
        exit(1)

    addr = socket.gethostbyname(sys.argv[1])
    iface = get_if()

    pkt = Ether(src=get_if_hwaddr(iface), dst="ff:ff:ff:ff:ff:ff")
    pkt = pkt / IP(dst=addr) / UDP(dport=4321, sport=1234) / str(time())
    # pkt.show2()
    sendp(pkt, iface=iface)
    i = 0
    try:
        while (True):
            if (i == 100):
                i = 0
                print('')
            sendp(pkt, iface=iface, verbose=False)
            print(".", end='')
            sys.stdout.flush()
            sleep(0.15)
            i += 1
    except KeyboardInterrupt:
        print('\nexiting')


if __name__ == '__main__':
    main()
