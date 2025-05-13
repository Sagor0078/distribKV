package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	shardAddrs      = flag.String("shard-addrs", "localhost:8080", "Comma-separated list of shard addresses")
	iterations      = flag.Int("iterations", 1000, "The number of iterations for writing")
	readIterations  = flag.Int("read-iterations", 10000, "The number of iterations for reading")
	concurrency     = flag.Int("concurrency", 1, "Number of goroutines to run in parallel")
	servers         []string
	httpClient      = &http.Client{Timeout: 5 * time.Second}
)

func shardIndex(key string, numShards int) int {
	h := 0
	for i := 0; i < len(key); i++ {
		h = int(key[i]) + 31*h
	}
	return h % numShards
}

func writeRand() (key string) {
	key = fmt.Sprintf("key-%d", rand.Intn(1000000))
	value := fmt.Sprintf("value-%d", rand.Intn(1000000))

	values := url.Values{}
	values.Set("key", key)
	values.Set("value", value)

	shard := shardIndex(key, len(servers))
	addr := servers[shard]

	resp, err := httpClient.Get("http://" + addr + "/set?" + values.Encode())
	if err != nil {
		log.Fatalf("Error during set: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	return key
}

func readRand(allKeys []string) (key string) {
	key = allKeys[rand.Intn(len(allKeys))]

	values := url.Values{}
	values.Set("key", key)

	shard := shardIndex(key, len(servers))
	addr := servers[shard]

	resp, err := httpClient.Get("http://" + addr + "/get?" + values.Encode())
	if err != nil {
		log.Fatalf("Error during get: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	return key
}

func benchmark(name string, iter int, fn func() string) (qps float64, strs []string) {
	var max time.Duration
	var min = time.Hour
	start := time.Now()

	for i := 0; i < iter; i++ {
		iterStart := time.Now()
		strs = append(strs, fn())
		iterTime := time.Since(iterStart)

		if iterTime > max {
			max = iterTime
		}
		if iterTime < min {
			min = iterTime
		}
	}

	total := time.Since(start)
	qps = float64(iter) / total.Seconds()
	avg := total / time.Duration(iter)
	fmt.Printf("Func %s took %s avg, %.1f QPS, %s max, %s min\n", name, avg, qps, max, min)
	return qps, strs
}

func benchmarkWrite() (allKeys []string) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalQPS float64

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			qps, keys := benchmark("write", *iterations, writeRand)
			mu.Lock()
			allKeys = append(allKeys, keys...)
			totalQPS += qps
			mu.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	log.Printf("Write total QPS: %.1f, keys written: %d", totalQPS, len(allKeys))
	return allKeys
}

func benchmarkRead(allKeys []string) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalQPS float64

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			qps, _ := benchmark("read", *readIterations, func() string {
				return readRand(allKeys)
			})
			mu.Lock()
			totalQPS += qps
			mu.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	log.Printf("Read total QPS: %.1f", totalQPS)
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	servers = strings.Split(*shardAddrs, ",")

	fmt.Printf("Benchmarking with %d write iterations, %d read iterations, %d concurrency, on %d shards\n",
		*iterations, *readIterations, *concurrency, len(servers))

	allKeys := benchmarkWrite()
	benchmarkRead(allKeys)
}
