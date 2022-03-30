p4program = simple
p4program-alt = asymmetric

all: debug
debug:
	go run . -n 3 -program $(p4program) -program-alt $(p4program-alt) -verbose
prod:
	go run . -n 3 -program $(p4program) -program-alt $(p4program-alt)

