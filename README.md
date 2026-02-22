# GopherTick

GopherTick is a high-frequency data aggregator and WebSocket broadcaster built in Go. This project measures the performance limits of a single-threaded broadcast hub on consumer-grade hardware.

## Technical Challenge
The system is designed to ingest data from multiple asynchronous sources at 100ms intervals, merge them into a single master stream, and broadcast them to thousands of concurrent WebSocket clients. The primary engineering goal was to implement a non-blocking fan-out that prevents slow network consumers from lagging the entire system.

## Architectural Features
*   **Fan-In Mixer:** Uses goroutines and a central channel to aggregate $N$ independent data providers into a unified feed.
*   **Interface-Driven Providers:** Decouples data generation from the broadcast logic for extensibility.
*   **Non-blocking Broadcast:** Implements a `select/default` pattern in the Hub. If a client's buffer is full, the current packet is dropped to maintain real-time performance for the rest of the pool.
*   **Single-Hub Model:** Intentionally uses a single-threaded broadcast loop to identify the exact point of CPU core saturation.

## Performance Benchmarks (Intel MacBook Pro)

Using the internal `cmd/loadtest` tool, the following metrics were captured during a high-stress run:

*   **Peak Throughput:** 361,752 messages per second.
*   **Total Data Delivered:** 10.8 Million JSON packets in 30 seconds.
*   **Concurrent Connections:** ~9,100 successful WebSocket sessions.
*   **Memory Efficiency:** ~32.01 KB per connection (Total ~283MB for the entire test).
*   **Average Handshake Latency:** 756.65 ms.

### Data Analysis & Bottlenecks
1.  **OS Limit:** The system hit a hard ceiling at **10,240 open files**. This is a macOS kernel restriction (`ulimit`) that prevented scaling to 10,000+ users.
2.  **Latency Saturation:** The average handshake latency of **756ms** indicates that at 9,000+ concurrent users, the single-threaded Hub loop has reached its processing limit. While throughput remains high, the time required to iterate through the client map begins to delay the processing of new connection handshakes.
3.  **RAM vs. CPU:** The test proved that memory is not the bottleneck (~4KB raw goroutine cost); rather, the system is CPU-bound by the serialization and iteration overhead of the single Hub.

## Requirements
*   Go 1.25+
*   `gorilla/websocket`
*   `sqlx` (for future persistence layers)

## How to Run
1.  **Start the Server:**
    ```bash
    go run cmd/gophertick/main.go
    ```
2.  **Run the Diagnostic & Load Test:**
    ```bash
    # Note: Requires ulimit adjustment to reach 10k
    ulimit -n 10000
    go run cmd/loadtest/main.go -n 9900 -ramp 15
    ```
