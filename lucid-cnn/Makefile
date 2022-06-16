SHELL=/bin/bash
CONDAROOT=/home/prince7/miniconda3

#attack:
#	./attack.sh
lucid-conda:
	bash $(CONDAROOT)/bin/activate python38
	./lucid.sh
	bash $(CONDAROOT)/bin/deactivate 
checkip:
	curl http://localhost:10000/digests/0 > /tmp/dump.json
	cat /tmp/dump.json | jq -r '.[] | .src_ip + " " + .dst_ip' | sort -u
