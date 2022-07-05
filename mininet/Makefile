p4program = asymmetric
topology = topology.json
singlesw = singlesw-topo.json
3sw = 3sw-topology.json

1all: mininet
singlesw:
	sudo env/bin/python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology ../config/$(singlesw)
3sw:
	sudo env/bin/python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology ../config/$(3sw)
mininet:
	sudo env/bin/python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology ../config/$(topology)
tls:
	cp ../cert/*.pem /tmp
	sudo env/bin/python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology ../config/$(topology) --cert-file /tmp/cert.pem --key-file /tmp/key.pem
stop:
	sudo mn -c
