# ulissep4controller

## Build step

Compile from source PI/bmv2

1.  Follow instructions in the [PI README](https://github.com/p4lang/PI#dependencies) to configure dependencies
1.  Configure, build and install PI:
    ```bash
    ./autogen.sh
    ./configure  --with-proto --with-cli
    make -j$(nproc)
    sudo make install && sudo ldconfig
    ```
1.  Follow instructions in the [bmv2 README](https://github.com/p4lang/behavioral-model/blob/main/README.md) to configure dependencies
1.  Configure and build the bmv2 code; from the root of the repository:
    ```bash
    ./autogen.sh
    ./configure --with-pi --without-nanomsg --disable-logging-macros --disable-elogger 'CXXFLAGS=-g -O3' 'CFLAGS=-g -O3'
    make -j$(nproc)
    sudo make install && sudo ldconfig
    ```

## How to run it

1. Start mininet
```bash
make topo
```

2. Start the controller (in another terminal)
```bash
make ctrl
```

3. Start sending packets (from mininet)
```bash
mininet> h1 iperf3 -c h2 -u -R -t 720
```

### Useful links

- https://github.com/antoninbas/p4runtime-go-client
- https://gitlab.com/wild_boar/netprog_course/-/tree/master/P4lab/exercises/5_asymmetric_flow
- https://gitlab.com/wild_boar/netprog_course/-/tree/master/P4lab/exercises/1_basic
