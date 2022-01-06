# TODO for implementing TLS

## TLS Server in P4

```cpp
    grpc::experimental::IdentityKeyCertPair keyPair = {"cert.pem", "key.pem"};
    auto certProvider = grpc::experimental::StaticDataCertificateProvider(keyPair);
    auto tlsOptions = grpc::experimental::TlsServerCredentialsOptions(certProvider);
    auto serverCredentials = grpc::experimental::TlsServerCredentials(tlsOptions);
    // auto serverCredentials = grpc::InsecureServerCredentials();
    builder.AddListeningPort(dp_grpc_server_addr, serverCredentials, &dp_grpc_server_port);
```

### Resources

- [C++ ALTS](https://grpc.io/docs/languages/cpp/alts)
- [GO ALTS](https://grpc.io/docs/languages/go/alts/)
- [simple_switch_grpc server](https://github.com/p4lang/behavioral-model/blob/182810a20e6293ae72c06699f74106321b5cd83a/targets/simple_switch_grpc/switch_runner.cpp#L523)
- [main.go controller](https://github.com/alsadiamir/ulissep4controller/blob/622fe73325ae810757cd11b7e80f583480b31e8a/controller/main.go#L332)
- [C++ stack overflow](https://stackoverflow.com/questions/32792284/grpc-in-cpp-providing-tls-support)