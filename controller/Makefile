p4config = ../config/config.json
p4config-alt = ../config/config-alt.json

all: debug
debug:
	go run . -n 4 -config $(p4config) -config-alt $(p4config-alt) -verbose
trace:
	go run . -n 4 -config $(p4config) -config-alt $(p4config-alt) -trace
prod:
	go run . -n 4 -config $(p4config) -config-alt $(p4config-alt)

