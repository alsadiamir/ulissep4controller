p4program = p4_packet_management
p4program-alt = p4_packet_management
devices = 1

all: debug
debug:
	go run . -n $(devices) -program $(p4program) 
	#-program-alt $(p4program-alt) -verbose
#prod:
#	go run . -n $(devices) -program $(p4program) -program-alt $(p4program-alt)

