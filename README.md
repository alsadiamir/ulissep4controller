# ulissep4controller

## How to run it

1. Start mininet - in the control plane folder

```console
make topo
```

2. Start the controller (in another terminal) - in the control plane folder

```console
make ctrl
```

3. Start sending packets (from mininet) to simulate the attack - in the lucid folder

```console
./attack.sh
```

4. Hopefully see some digests in stdio - which I don't :(

### Useful links

- https://github.com/antoninbas/p4runtime-go-client
- https://gitlab.com/wild_boar/netprog_course/-/tree/master/P4lab/exercises/5_asymmetric_flow
- https://gitlab.com/wild_boar/netprog_course/-/tree/master/P4lab/exercises/1_basic
