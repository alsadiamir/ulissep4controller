
from lib.p4_mininet import P4Host, P4GrpcSwitch
import psutil
import subprocess
import argparse
import os
import sys
from mininet.log import setLogLevel
from mininet.net import Mininet
from mininet.topo import Topo
from mininet.cli import CLI
from mininet.link import TCLink


parser = argparse.ArgumentParser(description='Mininet demo')
parser.add_argument('--p4-file', help='Path to P4 file', type=str, action="store", required=False)
parser.add_argument('--delay', help='s1-s2 delay for OWD testing', type=str, action="store", required=False)


def get_all_virtual_interfaces():
    try:
        return str.decode(subprocess.check_output("ip addr | grep s.-eth. | cut -d\':\' -f2 | cut -d\'@\' -f1", shell=True)).split('\n')
    except subprocess.CalledProcessError as e:
        print('Cannot retrieve interfaces.')
        print(e)
        return ''


class SingleSwitchTopo(Topo):
    "Single switch connected to n (< 256) hosts."

    def __init__(self, sw_path, json_path, delay, **opts):
        # Initialize topology and default options
        Topo.__init__(self, **opts)
        s1 = self.addSwitch('s1', sw_path=sw_path, json_path=json_path, grpc_port=50051,
                            device_id=1, cpu_port='255')
        s2 = self.addSwitch('s2', sw_path=sw_path, json_path=json_path, grpc_port=50052,
                            device_id=2, cpu_port='255')
        server = self.addHost('ser', ip="10.10.3.3/16", mac='00:00:01:01:01:01')

        self.addLink(s1, s2, delay=delay)
        self.addLink(s2, server)
        self.addLink(s1, server)


def main():
    p4base = args.p4_file[:-3]
    p4info = p4base + ".p4info.txt"
    p4json = p4base + ".json"
    p4dir = "/".join(args.p4_file.split("/")[:-1])

    result = os.system(f'p4c --target bmv2 --arch v1model --p4runtime-files {p4info} -o {p4dir} {args.p4_file}')

    if result != 0:
        print("Error while compiling!")
        sys.exit()

    topo = SingleSwitchTopo("simple_switch_grpc",
                            p4json,
                            args.delay)

    net = Mininet(topo=topo,
                  host=P4Host,
                  switch=P4GrpcSwitch,
                  controller=None,
                  link=TCLink)
    net.start()

    switch_running = "simple_switch_grpc" in (p.name() for p in psutil.process_iter())
    if switch_running == False:
        print("The switch didnt start correctly! Check the path to your P4 file!!")
        sys.exit()

    print("Starting mininet!")
    CLI(net)


if __name__ == '__main__':
    args = parser.parse_args()
    setLogLevel('info')
    main()
