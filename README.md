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

### Useful links

- https://github.com/antoninbas/p4runtime-go-client
- https://gitlab.com/wild_boar/netprog_course/-/tree/master/P4lab/exercises/5_asymmetric_flow
- https://gitlab.com/wild_boar/netprog_course/-/tree/master/P4lab/exercises/1_basic
