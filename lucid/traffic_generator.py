import sys

sys.path.append('/home/prince7/miniconda3/envs/python38/lib/python3.8/site-packages')

import argparse
import subprocess
import re
import os
import time
import psutil

    
         
           
def main():
    args=parse_input()
    replay_pcap(args.pcap_file, args.pcap_replay_speed, args.interface, args.attack_duration)
     

def replay_pcap(pcap_file, speed, interface,duration):

    tend=time.time()+60*duration
    if not os.path.exists(pcap_file):
        print("Pcap file doesn't exist")
        sys.exit(-1)

  
    print("start replaing traffic at {} Mbps".format(speed))
    replay=subprocess.Popen(['/usr/bin/tcpreplay', '--intf1', interface, '--mbps', str(speed), pcap_file], stdout=subprocess.DEVNULL)
    replay = psutil.Process(replay.pid)

    while time.time() <= tend:
        time.sleep(2)
        if replay.status() == psutil.STATUS_ZOMBIE:
            print("start replaing traffic again at {} Mbps".format(speed))
            replay=subprocess.Popen(['/usr/bin/tcpreplay', '--intf1', interface, '--mbps', str(speed), pcap_file], stdout=subprocess.DEVNULL)
            replay = psutil.Process(replay.pid)

    replay.kill()
    print("Replay finished!")


def parse_input():
    class ParseEthernetAddress(argparse.Action):
        def __init__(self, option_strings, dest, nargs=None, **kwargs):
            if nargs is not None:
                raise ValueError("nargs not allowed")
            super().__init__(option_strings, dest, **kwargs)

        def __call__(self, parser, namespace, value, option_string=None):
            if re.match("[0-9a-f]{2}([:]?)[0-9a-f]{2}(\\1[0-9a-f]{2}){4}$", value.lower()):
                setattr(namespace, self.dest, value)
            else:
                raise argparse.ArgumentError(self, "Wrong Address format")


    parser = argparse.ArgumentParser(description='Replay pcap files for a specific period of time')

    parser.add_argument('-f','--pcap_file', type=str, help='Pcap file to replay')
    parser.add_argument('-s', '--pcap_replay_speed',type=int, default=10,help='Replay speed')
    parser.add_argument('-i', '--interface', type=str, help='Interface used to send traffic')
    parser.add_argument('-ad','--attack_duration', default=10, type=int, help='Duration of the attack in minutes')

   
    return parser.parse_args()



if __name__ ==  "__main__":
    main()

