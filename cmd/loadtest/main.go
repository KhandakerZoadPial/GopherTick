package main

import (
	"flag"
	"fmt"
	"net/url"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// Live metrics trackers
var (
	activeConnections int64
	failedConnections int64
	totalMessages     int64
	totalHandshakeMs  int64
)

func main() {
	// 1. Config Flags
	targetBots := flag.Int("n", 3000, "Number of concurrent bots to spawn")
	rampUpMs := flag.Int("ramp", 15, "Milliseconds delay between spawns")
	testDuration := flag.Int("duration", 30, "How long to run the test (in seconds) after ramping")
	flag.Parse()

	// 2. CHECK OS LIMITS (Pre-flight)
	var rLimit syscall.Rlimit
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	osFileLimit := rLimit.Cur

	fmt.Println("\n=================================================================")
	fmt.Println(" 🔍 GOPHERTICK : DYNAMIC CAPACITY & BENCHMARK TOOL")
	fmt.Println("=================================================================")
	fmt.Printf("\n[ ENVIRONMENT LIMITS ]\n")
	fmt.Printf("  • CPU Cores:                 %d\n", runtime.NumCPU())
	fmt.Printf("  • Active OS File Limit:      %d (ulimit -n)\n", osFileLimit)

	if int64(*targetBots) > int64(osFileLimit) {
		fmt.Printf("  ⚠️ WARNING: You are asking for %d bots, but OS only allows %d open files.\n", *targetBots, osFileLimit)
	}

	// 3. CAPTURE BASELINE MEMORY
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	baseMem := m1.Sys

	// 4. START THE ATTACK
	fmt.Printf("\n[ STARTING LOAD TEST ]\n")
	fmt.Printf("  • Target Connections:        %d\n", *targetBots)
	fmt.Printf("  • Ramp-up delay:             %d ms/bot\n", *rampUpMs)

	targetUrl := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	var wg sync.WaitGroup
	attackStart := time.Now()

	for i := 0; i < *targetBots; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Rate limiting to prevent OS Thread panic
			time.Sleep(time.Duration(id**rampUpMs) * time.Millisecond)

			dialStart := time.Now()
			c, _, err := websocket.DefaultDialer.Dial(targetUrl.String(), nil)
			if err != nil {
				atomic.AddInt64(&failedConnections, 1)
				return
			}
			defer c.Close()

			// Record successful connection
			atomic.AddInt64(&activeConnections, 1)
			atomic.AddInt64(&totalHandshakeMs, time.Since(dialStart).Milliseconds())

			// Listen for data
			for {
				if _, _, err := c.ReadMessage(); err != nil {
					atomic.AddInt64(&activeConnections, -1)
					return
				}
				atomic.AddInt64(&totalMessages, 1)
			}
		}(i)

		// Print progress
		if i%100 == 0 && i > 0 {
			fmt.Printf("\r  -> Spawning bots: %d/%d", i, *targetBots)
		}
	}

	fmt.Printf("\r  -> Spawning bots: %d/%d (Done)\n", *targetBots, *targetBots)

	// 5. THE MEASUREMENT WINDOW (FIXED MATH)
	rampUpTime := time.Duration(*targetBots**rampUpMs) * time.Millisecond
	fmt.Printf("  -> Waiting %v for all bots to connect...\n", rampUpTime)
	fmt.Printf("  -> Then collecting data for %d seconds...\n", *testDuration)

	// Wait for the ramp-up to finish, PLUS the actual test duration
	time.Sleep(rampUpTime + (time.Duration(*testDuration) * time.Second))

	// 6. CAPTURE FINAL METRICS
	runtime.ReadMemStats(&m1)
	peakMem := m1.Sys
	memUsed := peakMem - baseMem

	active := atomic.LoadInt64(&activeConnections)
	failed := atomic.LoadInt64(&failedConnections)
	msgs := atomic.LoadInt64(&totalMessages)

	var avgHandshake float64
	var memPerUser float64
	if active > 0 {
		avgHandshake = float64(atomic.LoadInt64(&totalHandshakeMs)) / float64(active)
		memPerUser = float64(memUsed) / float64(active) / 1024.0 // in KB
	}

	// Throughput is calculated based on the stable test window, ignoring the ramp-up chaos
	throughput := float64(msgs) / float64(*testDuration)
	totalTime := time.Since(attackStart)

	// 7. PRINT DYNAMIC REPORT
	fmt.Println("\n=================================================================")
	fmt.Println(" 📊 LIVE BENCHMARK RESULTS")
	fmt.Println("=================================================================")

	fmt.Printf("  • Total Test Duration:       %v\n", totalTime.Round(time.Second))
	fmt.Printf("  • Successful Connections:    %d/%d\n", active, *targetBots)
	fmt.Printf("  • Failed Connections:        %d\n", failed)
	fmt.Printf("  • Avg Handshake Latency:     %.2f ms\n\n", avgHandshake)

	fmt.Printf("  • Total Messages Received:   %d\n", msgs)
	fmt.Printf("  • Peak Throughput:           %.2f msgs/sec\n\n", throughput)

	fmt.Printf("  • Tester Memory Used:        %d MB\n", memUsed/1024/1024)
	fmt.Printf("  • Memory per Connection:     ~%.2f KB\n", memPerUser)

	fmt.Println("\n[ DIAGNOSIS ]")
	if failed > 0 {
		fmt.Printf("  ⚠️ Hit a wall. OS prevented %d connections.\n", failed)
	} else if active < int64(*targetBots) {
		fmt.Println("  ⚠️ Some bots didn't finish connecting. Ramp-up time was cut short.")
	} else {
		fmt.Println("  ✅ Clean Run. System handled the target load flawlessly.")
	}
	fmt.Println("=================================================================\n")
}
