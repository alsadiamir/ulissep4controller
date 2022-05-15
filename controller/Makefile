p4config = ../config/config.json
p4config-alt = ../config/config-alt.json
singlesw-config = ../config/singlesw-config.json
singlesw-alt = ../config/singlesw-config-alt.json


all: debug
singlesw:
	go run . -n 1 -config $(singlesw-config) -config-alt $(singlesw-alt) -trace
debug:
	go run . -n 4 -config $(p4config) -config-alt $(p4config-alt) -verbose
trace:
	go run . -n 4 -config $(p4config) -config-alt $(p4config-alt) -trace
prod:
	go run . -n 4 -config $(p4config) -config-alt $(p4config-alt)
tls:
	go run . -n 4 -config $(p4config) -config-alt $(p4config-alt) -cert-file /tmp/cert.pem -verbose

