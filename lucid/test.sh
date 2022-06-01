#!/bin/bash

start=$(date)
echo "start $start" >> times.txt
declare -a attack_speeds=(100 10 20 30 50 62 80 100)
export target_interface=s1-eth2 # bmv2 interface
export pcap_file=output.pcap #traffic trace 
export job_duration=6 #minutes
export register_bits=17x2 #bits 17 bits for 2 blocks
export register_width=262144 # 2x2^bits eg. 2*2^17
declare -a samplig_values=(1 2 3 5 8 10)
dataset_type=IDS2017
model=10t-10n-IDS2017-LUCID-p4-training-dimezzato.h5


#10t-10n-IDS2017-LUCID-p4.h5

#/mnt/23a9947e-eee5-4436-aa57-a376a6a011e4/GitRepo/master-thesis-ddos-detection-via-ml-and-programmable-data-planes/sample-dataset/10t-10n-SYN2020-LUCID.h5

for i in "${attack_speeds[@]}"
do
	for j in "${samplig_values[@]}"
	do
		attack_speed=$i
		sampling=$j
		export attack_speed
		export sampling
		echo "start lucid with attack at $attack_speed mbps (background speed is $background_speed mbps), with sampling at $sampling, 1 packet over $sampling"
		echo "register_write sampling_treshold 0 $sampling" | simple_switch_CLI --thrift-port 22222 # set sampling rate in switch
		# conda activate python38 ... conda env has to be already up
		python3 lucid_cnn.py --predict_live localhost:22222 --model $model  --dataset_type $dataset_type
		echo ""
		echo ""
		echo ""
		echo ""
		echo ""
		echo ""

		echo "reset registers in switch"

		echo "register_reset block_writing" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset block_of_registers" | simple_switch_CLI --thrift-port 22222
		sleep 1
        echo "register_reset sampling_counter" | simple_switch_CLI --thrift-port 22222
        sleep 1
		echo "register_reset time0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset packet_length0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset tcp_flags0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset udp_len0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset dst_ip0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset ip_flags0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset src_ip0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset tcp_len0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset dst_port0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset ip_upper_protocol0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset src_port0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset tcp_window_size0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset icmp_type0" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset tcp_ack0" | simple_switch_CLI --thrift-port 22222


		echo "register_reset time1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset packet_length1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset tcp_flags1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset udp_len1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset dst_ip1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset ip_flags1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset src_ip1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset tcp_len1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset dst_port1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset ip_upper_protocol1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset src_port1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset tcp_window_size1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset icmp_type1" | simple_switch_CLI --thrift-port 22222
		sleep 1
		echo "register_reset tcp_ack1" | simple_switch_CLI --thrift-port 22222

		echo "reset done"
	done
done


echo "end: $(date)"
echo "end: $(date)" >>  times.txt

