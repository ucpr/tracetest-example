# tracetest example

- [tracetest で Trace-based Testing に触れてみる](https://ucpr.dev/articles/intro_trace_based_test)

![image](https://github.com/user-attachments/assets/7c0991ce-5f23-49cf-9ac8-ab5d716e1878)


## requirement

- tracetest cli
  - https://docs.tracetest.io/getting-started/install-cli
- docker

## run

### start components

```bash
docker compose up
```

### run tracetest

configure tracetest cli to use the local server

```bash
tracetest configure --server-url http://localhost:11633
```

run test

```bash
tracetest run test -f tracetest/testspec.yaml
```
