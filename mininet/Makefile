p4program = simple
topology = topology.json
singlesw = singlesw-topo.json

1all: mininet
singlesw:
	cp ../cert/*.pem /tmp
	sudo env/bin/python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology ../config/$(singlesw)
mininet:
	cp ../cert/*.pem /tmp
	sudo env/bin/python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology ../config/$(topology)
stop:
	sudo mn -c
restart: stop mininet
