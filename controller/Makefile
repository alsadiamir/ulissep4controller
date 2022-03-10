p4program = simple

all: debug
debug:
	go run . -n 3 -bin ../p4/$(p4program).json -p4info ../p4/$(p4program).p4info.txt -verbose
prod:
	go run . -n 3 -bin ../p4/$(p4program).json -p4info ../p4/$(p4program).p4info.txt

