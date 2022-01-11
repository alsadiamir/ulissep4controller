# TLS Support

## Build step

Compile from source PI/bmv2

1.  Follow instructions in the [PI README](https://github.com/p4lang/PI#dependencies) to configure dependencies
1.  Configure, build and install PI:
    ```
    git apply ulissep4controller/PI.patch
    ./autogen.sh
    ./configure --with-proto --without-internal-rpc --without-cli --without-bmv2
    make -j$(nproc)
    sudo make install && sudo ldconfig
    ```
1.  Configure and build the bmv2 code; from the root of the repository:
    ```
    ./autogen.sh
    ./configure --with-pi --without-thrift --without-nanomsg
    make -j$(nproc)
    sudo make install && sudo ldconfig
    ```
1.  Configure and build the simple_switch_grpc code; from this directory:
    ```
    ./autogen.sh
    ./configure
    make -j$(nproc)
    sudo make install
    ```

### Process

- find where the grpc server is instantiate
- check for support for ssl in the cpp server
- [configure pi_server.cpp to support ssl](https://github.com/alsadiamir/ulissep4controller/blob/main/PI.patch)
- [configure client for using tls](https://github.com/alsadiamir/ulissep4controller/commit/5e9b422e85be565971019b4c34b6c20b0c95c4b5)
- [created cert.go to generate certificates](https://github.com/alsadiamir/ulissep4controller/blob/main/cert/cert.go)
- compilation error, solution --without-thrift
- try different version of grpc, why ssl and tls are compatible?
