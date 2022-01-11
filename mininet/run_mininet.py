
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


class BaseTopo(Topo):

    def __init__(self, sw_path, json_path, **opts):
        # Initialize topology and default options
        Topo.__init__(self, **opts)
        s1 = self.addSwitch('s1', sw_path=sw_path, json_path=json_path, grpc_port=50051,
                            device_id=1, cpu_port='255')
        s2 = self.addSwitch('s2', sw_path=sw_path, json_path=json_path, grpc_port=50052,
                            device_id=2, cpu_port='255')
        server = self.addHost('server', ip="10.10.3.3/16", mac='00:00:01:01:01:01')

        self.addLink(s1, s2, port1=1, port2=1)
        self.addLink(s1, server, port1=2, port2=1)


def main():
    p4base = args.p4_file[:-3]
    p4info = p4base + ".p4info.txt"
    p4json = p4base + ".json"
    p4dir = "/".join(args.p4_file.split("/")[:-1])

    result = os.system(f'p4c --target bmv2 --arch v1model --p4runtime-files {p4info} -o {p4dir} {args.p4_file}')
    if result != 0:
        print("Error while compiling!")
        sys.exit()

    net = Mininet(topo=BaseTopo("simple_switch_grpc", p4json),
                  host=P4Host,
                  switch=P4GrpcSwitch,
                  controller=None,
                  link=TCLink)
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
