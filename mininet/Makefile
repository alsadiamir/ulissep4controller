DELAY = 100ms
name = simple

mininet: stop
	sudo python3 run_mininet.py --p4-file $(name).p4 --delay $(DELAY)
stop:
	sudo mn -c