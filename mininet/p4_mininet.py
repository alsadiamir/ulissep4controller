# Copyright 2013-present Barefoot Networks, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

from mininet.node import Switch, Host
from mininet.log import info, error, debug
from mininet.moduledeps import pathCheck
import sys
import os
import tempfile
import socket


class P4Host(Host):
    def config(self, mac=None, ip=None,
               defaultRoute=None, lo='up', **params):
        r = Host.config(self, mac, ip, defaultRoute, **params)

        self.defaultIntf().rename("eth0")

        for off in ["rx", "tx", "sg"]:
            cmd = "/sbin/ethtool --offload eth0 %s off" % off
            self.cmd(cmd)

        # disable IPv6
        self.cmd("sysctl -w net.ipv6.conf.all.disable_ipv6=1")
        self.cmd("sysctl -w net.ipv6.conf.default.disable_ipv6=1")
        self.cmd("sysctl -w net.ipv6.conf.lo.disable_ipv6=1")

        return r

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
    device_id = 0

    def __init__(self, name, sw_path=None, json_path=None,
                 grpc_port=None,
                 pcap_dump=False,
                 log_console=False,
                 verbose=False,
                 device_id=None,
                 enable_debugger=False,
                 cpu_port=None,
                 cert_file="/tmp/cert.pem",
                 key_file="/tmp/key.pem",
                 **kwargs):
        Switch.__init__(self, name, **kwargs)
        assert(sw_path)
        assert(json_path)
        # make sure that the provided sw_path is valid
        pathCheck(sw_path)
        # make sure that the provided JSON file exists
        if not os.path.isfile(json_path):
            error("Invalid JSON file.\n")
            sys.exit(1)
        self.sw_path = sw_path
        self.json_path = json_path
        self.verbose = verbose
        self.grpc_port = grpc_port
        self.cpu_port = cpu_port
        self.pcap_dump = pcap_dump
        self.enable_debugger = enable_debugger
        self.log_console = log_console
        self.logfile = "/tmp/p4s.{}.log".format(self.name)
        if device_id is not None:
            self.device_id = device_id
            P4GrpcSwitch.device_id = max(P4GrpcSwitch.device_id, device_id)
        else:
            self.device_id = P4GrpcSwitch.device_id
            P4GrpcSwitch.device_id += 1
        self.cert_file = cert_file
        self.key_file = key_file

    @classmethod
    def setup(cls):
        pass

    def check_switch_started(self, pid):
        """While the process is running (pid exists), we check if the Thrift
        server has been started. If the Thrift server is ready, we assume that
        the switch was started successfully. This is only reliable if the Thrift
        server is started at the end of the init process"""
        while True:
            if not os.path.exists(os.path.join("/proc", str(pid))):
                return False
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            try:
                sock.settimeout(0.5)
                result = sock.connect_ex(("localhost", self.grpc_port))
            finally:
                sock.close()
            if result == 0:
                return True

    def start(self, _):
        "Start up a new P4 switch"
        info("Starting P4 switch {}.\n".format(self.name))
        args = [self.sw_path]
        for port, intf in self.intfs.items():
            if not intf.IP():
                args.extend(['-i', str(port) + "@" + intf.name])
        if self.pcap_dump:
            args.append("--pcap")
        args.extend(['--device-id', str(self.device_id)])
        P4GrpcSwitch.device_id += 1
        args.append(self.json_path)
        if self.enable_debugger:
            args.append("--debugger")
        if self.log_console:
            args.append("--log-console")
        if self.grpc_port:
            grpc_args = ["--", "--grpc-server-addr", "0.0.0.0:"+str(self.grpc_port), "--cpu-port", self.cpu_port]
            grpc_args.extend(["--grpc-server-ssl", "--grpc-server-cert",
                             self.cert_file, "--grpc-server-key", self.key_file])
            args.extend(grpc_args)
        info(' '.join(args) + "\n")

        pid = None
        with tempfile.NamedTemporaryFile() as f:
            # self.cmd(' '.join(args) + ' > /dev/null 2>&1 &')
            self.cmd(' '.join(args) + ' >' + self.logfile + ' 2>&1 & echo $! >> ' + f.name)
            pid = int(f.read())
        debug("P4 switch {} PID is {}.\n".format(self.name, pid))
        # if not self.check_switch_started(pid):
        #     error("P4 switch {} did not start correctly.\n".format(self.name))
        #     sys.exit(1)
        info("P4 switch {} has been started.\n".format(self.name))

    def stop(self, _=True):
        "Terminate P4 switch."
        self.cmd('kill %' + self.sw_path)
        self.cmd('wait')
        self.deleteIntfs()

    def attach(self, _):
        "Connect a data port"
        assert(0)

    def detach(self, _):
        "Disconnect a data port"
        assert(0)
