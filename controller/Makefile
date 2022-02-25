p4program = asymmetric

all: ctrl
ctrl:
	go run . -n 3 -bin ../p4/$(p4program).json -p4info ../p4/$(p4program).p4info.txt -verbose
prod:
	go run . -n 3 -bin ../p4/$(p4program).json -p4info ../p4/$(p4program).p4info.txt

