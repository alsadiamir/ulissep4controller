# Copyright 2017-present Barefoot Networks, Inc.
# Copyright 2017-present Open Networking Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

import psutil
from mininet.node import Switch, Host
from mininet.log import info, error
from mininet.moduledeps import pathCheck
import sys
import os

SWITCH_START_TIMEOUT = 10  # seconds


def check_listening_on_port(port):
    for c in psutil.net_connections(kind='inet'):
        if c.status == 'LISTEN' and c.laddr[1] == port:
            return True
    return False


class P4Host(Host):
    def config(self, mac=None, ip=None,
               defaultRoute=None, lo='up', **_params):
        Host.config(self, mac, ip, defaultRoute, lo, **_params)

        self.defaultIntf().rename("eth0")

        for off in ["rx", "tx", "sg"]:
            cmd = "/sbin/ethtool --offload eth0 %s off" % off
            self.cmd(cmd)

        # disable IPv6
        self.cmd("sysctl -w net.ipv6.conf.all.disable_ipv6=1")
        self.cmd("sysctl -w net.ipv6.conf.default.disable_ipv6=1")
        self.cmd("sysctl -w net.ipv6.conf.lo.disable_ipv6=1")

    def describe(self):
        print("**********")
        print(self.name)
        print("default interface: %s\t%s\t%s" % (
            self.defaultIntf().name,
            self.defaultIntf().IP(),
            self.defaultIntf().MAC()
        ))
        print("**********")


class P4GrpcSwitch(Switch):
    """P4 virtual switch"""

    def __init__(self, name, sw_path=None, json_path=None,
                 grpc_port=None,
                 device_id=None,
                 cpu_port=None,
                 **kwargs):
        Switch.__init__(self, name, **kwargs)
        assert (sw_path)
        pathCheck(sw_path)
        self.sw_path = sw_path

        if json_path is not None:
            # make sure that the provided JSON file exists
            if not os.path.isfile(json_path):
                error("Invalid JSON file: {}\n".format(json_path))
                sys.exit(1)
            self.json_path = json_path
        self.grpc_port = grpc_port
        self.cpu_port = cpu_port
        self.device_id = device_id

    @classmethod
    def setup(cls):
        pass

    def start(self, _):
        "Start up a new P4 switch"
        info("\nStarting P4 switch %s\n" % self.name)
        args = [self.sw_path]
        for port, intf in self.intfs.items():
            if not intf.IP():
                args.extend(['-i', str(port) + "@" + intf.name])

        args.extend(['--device-id', str(self.device_id)])
        args.extend(["--log-flush", "--log-level", "debug", "--log-file", "log/%s.log" % self.name])

        if self.json_path:
            args.append(self.json_path)
        else:
            args.append("--no-p4")
        if self.grpc_port:
            args.append("-- --grpc-server-addr 0.0.0.0:"+str(self.grpc_port)+" --cpu-port "+self.cpu_port)

        info(' '.join(args))
        self.cmd(' '.join(args) + ' > log/%s.log 2>&1 &' % self.name)

        info("\nP4 switch %s has been started\n" % self.name)

    def stop(self, deleteIntfs=True):
        "Terminate IVS switch."
        Switch.stop(self, deleteIntfs)
        self.cmd('kill %' + self.sw_path)
        self.cmd('wait')
        self.deleteIntfs()

    def attach(self, _):
        "Connect a data port"
        assert(0)

    def detach(self, _):
        "Disconnect a data port"
        assert(0)
