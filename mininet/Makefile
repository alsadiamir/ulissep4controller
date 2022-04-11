p4program = simple

mininet:
	#stop
	cp ../cert/*.pem /tmp
	rm -rf log/*
	sudo env/bin/python3 run_mininet.py --p4-file ../p4/$(p4program).p4
stop:
	sudo mn -c
