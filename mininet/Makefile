name = simple

mininet: stop
	rm -rf log/*
	sudo env/bin/python3 run_mininet.py --p4-file ../p4/$(name).p4
stop:
	sudo mn -c 
	