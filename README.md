***
# GopherTick 

GopherTick is a high-frequency data aggregator and real-time broadcaster I built to explore the limits of Go's concurrency model. The project focuses on the "Broadcast Problem": how to ingest multiple high-speed data streams and push them to thousands of users simultaneously without the system locking up.

## The Motivation
Coming from a Python and Django background, I was used to frameworks that handle the heavy lifting for you. However, those frameworks often struggle with "chatty" I/O—scenarios where data updates 10-20 times per second. Scaling that in Python usually requires a complex stack of Redis, Celery, and multiple load balancers.

I wanted to see if I could achieve better results using only Go's native primitives (Goroutines and Channels) running on a single, consumer-grade machine.

## The Architecture
The system is built on a "Fan-In/Fan-Out" design:
*   **The Mixer (Fan-In):** I designed a concurrent engine that pulls data from multiple independent "Producers" (simulating live price feeds). Each producer runs in its own goroutine, and their output is merged into one master channel.
*   **The Hub (Fan-Out):** A centralized coordinator manages all active WebSocket connections. It clones every incoming data packet and distributes it to every connected user.
*   **Slow-Consumer Protection:** This was a critical architectural lesson. I implemented a non-blocking `select/default` pattern for broadcasts. If a user’s network is slow and their buffer fills up, the system drops their specific packet instead of making the entire server wait. This ensures the stream remains "real-time" for healthy clients.

## The Performance Curve (Intel MacBook Pro)
I didn't just want to see it work; I wanted to see it break. I built a custom, rate-limited load-testing tool (`cmd/loadtest`) to profile the system across different scales. These results were captured on a 4-core Intel MacBook Pro.

| Connections | Avg Handshake Latency | Peak Throughput | Verdict |
| :--- | :--- | :--- | :--- |
| 1,000 | 1.55 ms | 27,171 msgs/sec | Perfectly smooth |
| 3,000 | 8.18 ms | 136,887 msgs/sec | Perfectly smooth |
| 5,000 | 26.21 ms | 268,669 msgs/sec | CPU core begins working |
| 7,000 | 332.56 ms | 342,122 msgs/sec | **The "Knee":** Latency spikes |
| 9,000 | **409.50 ms** | **423,965 msgs/sec** | **Peak Capacity** |
| 10,000 | 449.22 ms | 361,671 msgs/sec | **Saturated:** Throughput drops |

### Lessons from the "Wall"
My MacBook has a default file limit of **10,240**. Through these tests, I discovered three major system boundaries:
1.  **Memory is cheap:** Even with nearly 10,000 concurrent users, the entire server environment used less than **300MB of RAM**. Each user costs only about **29KB-32KB**.
2.  **The OS is the Boss:** I reached a point where my Go code was faster than my Mac's kernel settings. I had to manually tune `ulimit` and `kern.maxprocperuid` just to allow the Go runtime to keep up with the incoming connection requests.
3.  **Single-Core Saturation:** At 9,000 users, the system delivered **~424,000 messages every second**. At this point, the single-threaded Hub loop became the bottleneck. The loop took so long to iterate through all users that it started delaying the "handshake" of new users, causing the latency spike.

## How to Run
You can run the server and the benchmark tool yourself to see how your machine handles the load.

### 1. Start the server
```bash
go run cmd/gophertick/main.go
```

### 2. Run the Benchmark Tool
This tool profiles your OS limits and tracks live throughput and latency metrics.
```bash
# Increase session file limits to allow the test to scale
ulimit -n 10000

# Run the test with 5,000 bots and a 15ms ramp-up delay
go run cmd/loadtest/main.go -n 5000 -ramp 15
```

## Tech Stack
*   **Language:** Go (Goroutines, Channels, Atomic Counters)
*   **Web:** `gorilla/websocket`
*   **Systems:** macOS Kernel Tuning, Capacity Profiling

***
