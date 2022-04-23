p4config = config.json
p4config-alt = config-alt.json

all: debug
debug:
	go run . -n 4 -config $(p4config) -config-alt $(p4config-alt) -verbose
prod:
	go run . -n 4 -config $(p4config) -config-alt $(p4config-alt)

