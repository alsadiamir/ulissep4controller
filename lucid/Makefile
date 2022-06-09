attack:
	./attack.sh
lucid:
	conda activate python38
	./lucid.sh
checkip:
	curl http://localhost:10000/digests/0 > /tmp/dump.json
	cat /tmp/dump.json | jq -r '.[] | .src_ip + " " + .dst_ip' | sort -u
