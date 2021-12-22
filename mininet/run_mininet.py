
import sys
import os

sys.path.append("/home/prince7/.local/lib/python3.8/site-packages") #to include psutil..depends on the machine
#sys.path.append("/usr/lib/python3/dist-packages")
sys.path.append(
    os.path.join(os.path.dirname(os.path.abspath(__file__)),
                 'utils/'))


import random
import argparse
from time import sleep
import subprocess

import psutil

from mininet.net import Mininet
from mininet.topo import Topo
from mininet.log import setLogLevel, info
from mininet.cli import CLI
from mininet.link import TCLink
from mininet.node import OVSSwitch, Ryu
from p4_mininet import P4Switch, P4Host, P4GrpcSwitch, P4OVSSwitch
from p4runtime_switch import P4RuntimeSwitch


parser = argparse.ArgumentParser(description='Mininet demo')
parser.add_argument('--num-hosts', help='Number of hosts to connect to switch', type=int, action="store", default=2)
parser.add_argument('--p4-file', help='Path to P4 file', type=str, action="store", required=False)
parser.add_argument('--delay', help='s1-s2 delay for OWD testing', type=str, action="store", required=False)

def get_all_virtual_interfaces():
    args = parser.parse_args()
    try:
        return str.decode(subprocess.check_output("ip addr | grep s.-eth. | cut -d\':\' -f2 | cut -d\'@\' -f1", shell=True)).split('\n')
    except subprocess.CalledProcessError as e:
        print_error('Cannot retrieve interfaces.')
        print_error(e)
        return ''

class SingleSwitchTopo(Topo):
    "Single switch connected to n (< 256) hosts."
    def __init__(self, sw_path, json_path, n, delay, **opts):
        # Initialize topology and default options
        Topo.__init__(self, **opts)
        s1 = self.addSwitch('s1', sw_path = sw_path, json_path = json_path, grpc_port = 50051, device_id = 1, cpu_port='255', thrift_port=9090)
        s2 = self.addSwitch('s2', sw_path = sw_path, json_path = json_path, grpc_port = 50052, device_id = 2, cpu_port='255', thrift_port=9091)
        
        self.addLink(s1, s2, delay=delay)

        for h in range(n):
            host = self.addHost('h%d' % (h + 1), ip = "10.10.10.%d/16" % (h + 1), mac = '00:04:00:00:00:%02x' %h)
            self.addLink(s1, host)
        server =  self.addHost('ser', ip = "10.10.3.3/16", mac = '00:00:01:01:01:01')    
        self.addLink(s2, server)
        #s1o = self.addSwitch('s1o', cls=OVSSwitch)
        #self.addLink(s1, s1o, port=111)

def main():
    num_hosts = int(args.num_hosts)
    sourceFileName=p4_file = args.p4_file.split('.')[0]

    result = os.system("p4c --target bmv2 --arch v1model --p4runtime-files p4files/"+sourceFileName+".p4info.txt -o p4files "+ args.p4_file) 
    p4_file = args.p4_file.split('/')[-1]
    json_file = "p4files/"+p4_file.split('.')[0] + ".json"

    topo = SingleSwitchTopo("simple_switch_grpc",
                            json_file,
                            num_hosts,
                            args.delay)

    net = Mininet(topo = topo,
                  host = P4Host,
                  switch = P4GrpcSwitch,
                  controller = None,
                  link = TCLink)
    net.start()

    
    if result !=0:
        print("Error while compiling!")
        exit()

    switch_running="simple_switch_grpc" in (p.name() for p in psutil.process_iter())
    if switch_running==False:    
        print("The switch didnt start correctly! Check the path to your P4 file!!")
        exit()

    print("Starting mininet!")

    CLI(net)

if __name__ == '__main__':
    args = parser.parse_args()
    setLogLevel( 'info' )
    main()
