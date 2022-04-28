
import json
from time import sleep
import argparse
import os
import sys
from mininet.log import setLogLevel
from mininet.net import Mininet
from mininet.topo import Topo
from mininet.cli import CLI
from p4_mininet import P4GrpcSwitch, P4Host


class TopoRunner:

    def __init__(self, topo_file, switch_json, bmv2_exe, cert_file, key_file):
        """ Initializes some attributes and reads the topology json. Does not
            actually run the exercise. Use run_exercise() for that.

            Arguments:
                topo_file : string    // A json file which describes the mininet topology.
                switch_json : string  // Path to a compiled p4 json for bmv2
                bmv2_exe    : string  // Path to the p4 behavioral binary
        """
        with open(topo_file, 'r', encoding='utf-8') as f:
            topo = json.load(f)
        self.hosts = topo['hosts']
        switches = topo['switches']
        links = self.parse_links(topo['links'])

        topo = P4Topo(self.hosts, switches, links, switch_json, bmv2_exe, cert_file, key_file)
        self.net = Mininet(topo=topo,
                           host=P4Host,
                           switch=P4GrpcSwitch,
                           controller=None)

    def parse_links(self, unparsed_links):
        """ Given a list of links descriptions of the form [node1, node2, latency, bandwidth]
            with the latency and bandwidth being optional, parses these descriptions
            into dictionaries and store them as self.links
        """
        links = []
        for link in unparsed_links:
            # make sure each link's endpoints are ordered alphabetically
            s, t, = link[0], link[1]
            if s > t:
                s, t = t, s

            link_dict = {'node1': s, 'node2': t}

            if link_dict['node1'][0] == 'h':
                assert link_dict['node2'][0] == 's', 'Hosts should be connected to switches, not ' + \
                    str(link_dict['node2'])
            links.append(link_dict)
        return links

    def program_hosts(self):
        """ Execute any commands provided in the topology.json file on each Mininet host
        """
        for host_name, host_info in list(self.hosts.items()):
            h = self.net.get(host_name)
            if "commands" in host_info:
                for cmd in host_info["commands"]:
                    h.cmd(cmd)

    def run_topology(self):
        """ Sets up the mininet instance, programs the switches,
            and starts the mininet CLI. This is the main method to run after
            initializing the object.
        """
        # Initialize mininet with the topology specified by the config
        self.net.start()
        sleep(1)

        # some programming that must happen after the net has started
        self.program_hosts()
        sleep(1)

        CLI(self.net)
        # stop right after the CLI is exited
        self.net.stop()


class P4Topo(Topo):

    """ The mininet topology class for the P4 tutorial exercises"""

    def __init__(self, hosts, switches, links, json_path, bmv2_exe, cert_file, key_file, **opts):
        Topo.__init__(self, **opts)
        host_links = []
        switch_links = []

        # assumes host always comes first for host<-->switch links
        for link in links:
            if link['node1'][0] == 'h':
                host_links.append(link)
            else:
                switch_links.append(link)

        for i, sw in enumerate(switches, start=1):
            self.addSwitch(sw, sw_path=bmv2_exe, json_path=json_path, grpc_port=50050+i, device_id=i, cpu_port='255',
             cert_file=cert_file, key_file=key_file)

        for link in host_links:
            host_name = link['node1']
            sw_name, sw_port = self.parse_switch_node(link['node2'])
            host_ip = hosts[host_name]['ip']
            host_mac = hosts[host_name]['mac']
            self.addHost(host_name, ip=host_ip, mac=host_mac)
            self.addLink(host_name, sw_name, port2=sw_port)

        for link in switch_links:
            sw1_name, sw1_port = self.parse_switch_node(link['node1'])
            sw2_name, sw2_port = self.parse_switch_node(link['node2'])
            self.addLink(sw1_name, sw2_name, port1=sw1_port, port2=sw2_port)

    def parse_switch_node(self, node):
        assert(len(node.split('-')) == 2)
        sw_name, sw_port = node.split('-')
        try:
            sw_port = int(sw_port[1:])
        except Exception as err:
            raise Exception('Invalid switch node in topology file: {}'.format(node)) from err
        return sw_name, sw_port


def main():
    p4base = args.p4_file[:-3]
    p4info = p4base + ".p4info.txt"
    p4json = p4base + ".json"
    p4dir = "/".join(args.p4_file.split("/")[:-1])

    topology = args.topology
    cert_file = args.cert_file
    key_file= args.key_file

    os.remove(f'{p4dir}/{p4json}')
    os.remove(f'{p4dir}/{p4info}')
    result = os.system(f'p4c --target bmv2 --arch v1model --p4runtime-files {p4info} -o {p4dir} {args.p4_file}')
    if result != 0:
        print("Error while compiling!")
        sys.exit()

    runner = TopoRunner(topology, p4json, 'simple_switch_grpc', cert_file, key_file)
    runner.run_topology()


parser = argparse.ArgumentParser(description='Mininet demo')
parser.add_argument('--p4-file', help='Path to P4 file', type=str, action="store", required=False)
parser.add_argument('--topology', help='Topology file', type=str, action="store", required=False)
parser.add_argument('--cert-file', help='Cert file for tls', type=str, action="store", required=False)
parser.add_argument('--key-file', help='Key file for tls', type=str, action="store", required=False)

if __name__ == '__main__':
    args = parser.parse_args()
    setLogLevel('info')
    main()
