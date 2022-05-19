# TLS Support

### Process

- find where the grpc server is instantiate
- check for support for ssl in the cpp server
- [configure pi_server.cpp to support ssl](https://github.com/alsadiamir/ulissep4controller/blob/main/PI.patch)
- [configure client for using tls](https://github.com/alsadiamir/ulissep4controller/commit/5e9b422e85be565971019b4c34b6c20b0c95c4b5)
- [created cert.go to generate certificates](https://github.com/alsadiamir/ulissep4controller/blob/main/cert/cert.go)
- compilation error, solution --without-thrift
- try different version of grpc, why ssl and tls are compatible?
- was submitted a commit to pi that made ssl available, a fix is needed

## Endnote

After the reasearch the dev team at p4 implemented native support for tls
