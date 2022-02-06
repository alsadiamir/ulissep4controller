# ulissep4controller

## How to run it

1. Start mininet

```console
make topo
```

2. Start the controller (in another terminal)

```console
make ctrl
```

3. Start sending packets (from mininet)

```console
h1 ./send.py 10.0.1.2
```
