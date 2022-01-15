
import psutil
import argparse
import os
import sys
from mininet.log import setLogLevel
from mininet.net import Mininet
from mininet.topo import Topo
from mininet.cli import CLI
from mininet.link import TCLink
from p4_mininet import P4GrpcSwitch, P4Host

parser = argparse.ArgumentParser(description='Mininet demo')
parser.add_argument('--p4-file', help='Path to P4 file', type=str, action="store", required=False)


class Topo4Switch(Topo):

    def __init__(self, sw_path, json_path, **opts):
        # Initialize topology and default options
        Topo.__init__(self, **opts)
        s1 = self.addSwitch('s1', sw_path=sw_path, json_path=json_path, grpc_port=50051,
                            device_id=1, cpu_port='255')
        s2 = self.addSwitch('s2', sw_path=sw_path, json_path=json_path, grpc_port=50052,
                            device_id=2, cpu_port='255')
        s3 = self.addSwitch('s3', sw_path=sw_path, json_path=json_path, grpc_port=50053,
                            device_id=3, cpu_port='255')
        s4 = self.addSwitch('s4', sw_path=sw_path, json_path=json_path, grpc_port=50054,
                            device_id=4, cpu_port='255')
        h1 = self.addHost('h1', ip="10.0.1.1/32", mac='00:00:01:01:01:01')
        h2 = self.addHost('h2', ip="10.0.1.2/32", mac='00:00:01:01:01:02')
        h3 = self.addHost('h3', ip="10.0.1.3/32", mac='00:00:01:01:01:03')
        h4 = self.addHost('h4', ip="10.0.1.4/32", mac='00:00:01:01:01:04')
        # h1 - s1 - s2 - h2
        #       |   |
        # h3 - s3 - s4 - h4
        # P1
        self.addLink(h1, s1)
        self.addLink(h2, s2)
        self.addLink(h3, s3)
        self.addLink(h4, s4)
        # P2
        self.addLink(s1, s2)
        self.addLink(s3, s4)
        # P3
        self.addLink(s1, s3)
        self.addLink(s2, s4)


def main():
    p4base = args.p4_file[:-3]
    p4info = p4base + ".p4info.txt"
    p4json = p4base + ".json"
    p4dir = "/".join(args.p4_file.split("/")[:-1])

    result = os.system(f'p4c --target bmv2 --arch v1model --p4runtime-files {p4info} -o {p4dir} {args.p4_file}')
    if result != 0:
        print("Error while compiling!")
        sys.exit()

    net = Mininet(topo=Topo4Switch("simple_switch_grpc", p4json),
                  host=P4Host,
                  switch=P4GrpcSwitch,
                  controller=None,
                  link=TCLink)
    net.xterms = True
    net.start()

    switch_running = "simple_switch_grpc" in (p.name() for p in psutil.process_iter())
    if switch_running == False:
        print("The switch didnt start correctly! Check the path to your P4 file!!")
        sys.exit(1)

    print("Starting mininet!")
    CLI(net)


if __name__ == '__main__':
    args = parser.parse_args()
    setLogLevel('info')
    main()
