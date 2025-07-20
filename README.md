# Caching-Proxy

### We use in-memory cache to achieve our goal

![Terminal Output](assets/screenshots/terminalOutput.png)

#### Steps to test:

1. Run `go build` to create the binary
2. Run `./caching-proxy --port 2020 --origin https://dummyjson.com` (an example command)
3. Open another terminal, and Run `curl - s http://localhost:2020/products/1 | jq` to get a pretty Response
4. NOTE: _https://localhost:x/_ won't work, (handshake cannot be established)
5. The server closes automatically after 10 minutes, providing graceful shutdown
6. To check headers, Run `curl -i http://localhost:2020/products/1`
7. Run `curl -s http://localhost:2020/shutdown` for manually closing the server

### Featues:

### Future Roadmap:

PROJECT LINK: https://roadmap.sh/projects/caching-server
