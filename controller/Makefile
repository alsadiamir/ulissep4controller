all: controller1 controller2
controller1:
	go run main.go --addr 127.0.0.1:50051 --device-id 1 --bin ../mininet/p4files/simple.json --p4info ../mininet/p4files/simple.p4info.txt -verbose
controller2:
	go run main.go --addr 127.0.0.1:50052 --device-id 2 --bin ../mininet/p4files/simple.json --p4info ../mininet/p4files/simple.p4info.txt

