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

import os
import psutil
from mininet.node import Switch, Host
from mininet.log import info, error, debug
from mininet.moduledeps import pathCheck
import sys
import tempfile
from time import sleep

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
    "BMv2 switch with gRPC support"
    next_grpc_port = 50051

    def __init__(self, name, device_id, sw_path=None, json_path=None,
                 grpc_port=None,
                 pcap_dump=False,
                 log_console=False,
                 verbose=False,
                 enable_debugger=False,
                 log_file=None,
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
        else:
            self.json_path = None

        if grpc_port is not None:
            self.grpc_port = grpc_port
        else:
            self.grpc_port = P4GrpcSwitch.next_grpc_port
            P4GrpcSwitch.next_grpc_port += 1

        if check_listening_on_port(self.grpc_port):
            error('%s cannot bind port %d because it is bound by another process\n' % (self.name, self.grpc_port))
            sys.exit(1)

        self.verbose = verbose
        self.pcap_dump = pcap_dump
        self.enable_debugger = enable_debugger
        self.log_console = log_console

        if log_file is not None:
            self.log_file = log_file
        else:
            self.log_file = "log/%s.log" % self.name
        self.device_id = device_id

    def check_switch_started(self, pid):
        for _ in range(SWITCH_START_TIMEOUT * 2):
            if not os.path.exists(os.path.join("/proc", str(pid))):
                return False
            if check_listening_on_port(self.grpc_port):
                return True
            sleep(0.5)
        return False

    def start(self, _):
        "Start up a new P4 switch"

        info("Starting P4 switch {}.\n".format(self.name))
        args = [self.sw_path]
        for port, intf in list(self.intfs.items()):
            if not intf.IP():
                args.extend(['-i', str(port) + "@" + intf.name])
        if self.pcap_dump:
            args.append("--pcap %s" % self.pcap_dump)
        args.extend(['--device-id', str(self.device_id)])
        if self.json_path:
            args.append(self.json_path)
        else:
            args.append("--no-p4")
        if self.enable_debugger:
            args.append("--debugger")
        if self.log_console:
            args.append("--log-console")
        else:
            args.append("--log-file %s" % self.log_file)
        args.append("--log-flush --log-level trace")

        if self.grpc_port:
            args.append("-- --grpc-server-addr 0.0.0.0:" + str(self.grpc_port))
        cmd = ' '.join(args)
        info(cmd + "\n")

        pid = None
        with tempfile.NamedTemporaryFile() as f:
            self.cmd(cmd + ' >' + self.log_file + ' 2>&1 & echo $! >> ' + f.name)
            pid = int(f.read())
        debug("P4 switch {} PID is {}.\n".format(self.name, pid))
        if not self.check_switch_started(pid):
            error("P4 switch {} did not start correctly.\n".format(self.name))
            sys.exit(1)
        info("P4 switch {} has been started.\n".format(self.name))

    def stop(self, _=True):
        "Terminate P4 switch."
        self.cmd('kill %' + self.sw_path)
        self.cmd('wait')
        self.deleteIntfs()
