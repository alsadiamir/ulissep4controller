p4program = simple
topology = topology.json

mininet:
	#stop
	cp ../cert/*.pem /tmp
	rm -rf log/*
	sudo env/bin/python3 run_mininet.py --p4-file ../p4/$(p4program).p4 --topology $(topology)
stop:
	sudo mn -c
