p4program = asymmetric
topology = topology.json
singlesw = singlesw-topo.json

all: singlesw
singlesw:
	sudo python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology ../config/$(singlesw)
mininet:
	sudo python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology ../config/$(topology)
tls:
	cp ../cert/*.pem /tmp
	sudo python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology ../config/$(topology) --cert-file /tmp/cert.pem --key-file /tmp/key.pem
stop:
	sudo mn -c
