DELAY = 100ms
name = simple

topo: 
	make -C ./mininet mininet

controller1:
	make -C ./controller controller1
controller2:
	make -C ./controller controller2
