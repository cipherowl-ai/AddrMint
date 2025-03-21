package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blocto/solana-go-sdk/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/crypto"
)

// Version information (can be overridden by build flags)
var version = "dev"

// Job represents a single address generation task
type Job struct {
	index   int
	seed    string
	network string
}

// Result represents the result of a job
type Result struct {
	index   int
	address string
}

// ProgressBar displays a visual progress bar
type ProgressBar struct {
	total     int
	current   int
	width     int
	lastPrint time.Time
	mu        sync.Mutex
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, width int) *ProgressBar {
	return &ProgressBar{
		total:     total,
		width:     width,
		lastPrint: time.Now().Add(-1 * time.Second), // Start immediately
	}
}

// Update updates the progress bar
func (pb *ProgressBar) Update(current int) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.current = current

	// Only update the display if enough time has passed (limit refresh rate)
	if time.Since(pb.lastPrint) < 100*time.Millisecond && current < pb.total {
		return
	}

	pb.lastPrint = time.Now()
	percent := float64(pb.current) / float64(pb.total)
	filled := int(percent * float64(pb.width))

	// Create the bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", pb.width-filled)

	// Show the progress bar
	fmt.Fprintf(os.Stderr, "\r[%s] %d/%d (%.2f%%) ", bar, pb.current, pb.total, percent*100)

	// If we're done, print a newline
	if pb.current >= pb.total {
		fmt.Fprintln(os.Stderr)
	}
}

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	network := flag.String("network", "", "Blockchain network (ethereum, bitcoin, solana)")
	count := flag.Int("count", 1, "Number of addresses to generate")
	seedInt := flag.Int64("seed", 0, "Random seed as integer (0 for random seed)")
	workers := flag.Int("workers", runtime.NumCPU(), "Number of worker goroutines")
	batchSize := flag.Int("batch-size", 1000, "Number of addresses to batch before reporting progress")
	outputBufferSize := flag.Int("output-buffer", 10000, "Size of the output buffer for results")
	outputFile := flag.String("output", "", "Output file path (default: stdout)")
	generateHash := flag.Bool("generate-hash", false, "Prefix each address with a SHA-256 hash (first 6 characters) and comma")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Fprintf(os.Stderr, "AddrMint v%s - High-performance blockchain address generator\n", version)
		os.Exit(0)
	}

	startTime := time.Now()

	// Print banner
	fmt.Fprintf(os.Stderr, "AddrMint v%s - Blockchain Address Generator\n", version)
	fmt.Fprintf(os.Stderr, "==========================================\n")

	// Validate network
	if *network == "" {
		log.Fatal("Network is required. Use --network ethereum|bitcoin|solana")
	}

	if *network != "ethereum" && *network != "bitcoin" && *network != "solana" {
		log.Fatal("Network must be ethereum, bitcoin, or solana")
	}

	// Prepare the initial seed
	var baseSeed string
	if *seedInt == 0 {
		// Generate random seed if not provided
		randBytes := make([]byte, 32)
		_, err := rand.Read(randBytes)
		if err != nil {
			log.Fatal("Failed to generate random seed:", err)
		}
		baseSeed = hex.EncodeToString(randBytes)
		fmt.Fprintf(os.Stderr, "Generated random seed\n")
	} else {
		// Use the provided integer seed
		baseSeed = strconv.FormatInt(*seedInt, 16)
		fmt.Fprintf(os.Stderr, "Using seed value: %d\n", *seedInt)
	}

	// Setup output file if specified
	var output *os.File
	var err error
	if *outputFile != "" {
		output, err = os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer output.Close()
		fmt.Fprintf(os.Stderr, "Writing results to %s\n", *outputFile)
	} else {
		output = os.Stdout
	}

	fmt.Fprintf(os.Stderr, "Generating %d %s addresses using %d workers\n", *count, *network, *workers)

	// Optimize number of workers based on count
	if *count < *workers {
		*workers = *count
		fmt.Fprintf(os.Stderr, "Adjusted number of workers to %d based on address count\n", *workers)
	}

	// Create a worker pool with optimized channel sizes for better throughput
	jobs := make(chan Job, *workers*2)
	results := make(chan Result, *outputBufferSize)

	// Start workers
	var wg sync.WaitGroup
	for w := 1; w <= *workers; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	// Start a goroutine to close the results channel when all jobs are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Create a job submission pool for better memory efficiency
	jobPool := &sync.Pool{
		New: func() interface{} {
			return &Job{}
		},
	}

	// Submit jobs in batches for better memory efficiency
	go func() {
		batchSubmitJobs(jobs, *count, baseSeed, *network, *batchSize, jobPool)
		close(jobs)
	}()

	// Create an efficient result collector with progress bar
	resultCollector := NewResultCollector(*count, *batchSize, output, *generateHash)

	// Create progress bar
	progressBar := NewProgressBar(*count, 50) // 50 characters wide

	// Process results
	for result := range results {
		resultCollector.AddResult(result, progressBar)
	}

	elapsedTime := time.Since(startTime)
	fmt.Fprintf(os.Stderr, "Generated %d addresses in %s (%.2f addresses/sec)\n",
		*count, elapsedTime, float64(*count)/elapsedTime.Seconds())
}

// batchSubmitJobs submits jobs in batches for better memory efficiency
func batchSubmitJobs(jobs chan<- Job, count int, baseSeed, network string, batchSize int, pool *sync.Pool) {
	for i := 0; i < count; i++ {
		// Modify seed for each iteration to get different addresses
		h := sha256.New()
		h.Write([]byte(baseSeed + fmt.Sprintf("%d", i)))
		seedValue := hex.EncodeToString(h.Sum(nil))

		// Get a job from the pool
		job := pool.Get().(*Job)
		job.index = i
		job.seed = seedValue
		job.network = network

		// Submit the job
		jobs <- *job

		// Put the job back in the pool
		pool.Put(job)
	}
}

// ResultCollector efficiently collects and prints results
type ResultCollector struct {
	resultMap    map[int]string
	resultCount  int
	nextToPrint  int
	totalCount   int
	batchSize    int
	mu           sync.Mutex
	outputFile   *os.File
	generateHash bool
}

// NewResultCollector creates a new result collector
func NewResultCollector(totalCount, batchSize int, outputFile *os.File, generateHash bool) *ResultCollector {
	return &ResultCollector{
		resultMap:    make(map[int]string),
		totalCount:   totalCount,
		batchSize:    batchSize,
		outputFile:   outputFile,
		generateHash: generateHash,
	}
}

// AddResult adds a result to the collector and prints results in order
func (rc *ResultCollector) AddResult(result Result, progressBar *ProgressBar) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.resultMap[result.index] = result.address
	rc.resultCount++

	// Update progress bar
	progressBar.Update(rc.resultCount)

	// Print results in order
	for {
		if address, exists := rc.resultMap[rc.nextToPrint]; exists {
			if rc.generateHash {
				// Generate a hash from the address
				h := sha256.New()
				h.Write([]byte(address))
				hash := hex.EncodeToString(h.Sum(nil))
				// Use first 6 characters of hash for shorter representation
				fmt.Fprintf(rc.outputFile, "%s,%s\n", hash[:6], address)
			} else {
				fmt.Fprintln(rc.outputFile, address)
			}
			delete(rc.resultMap, rc.nextToPrint)
			rc.nextToPrint++
		} else {
			break
		}
	}
}

func worker(id int, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		var address string

		switch job.network {
		case "ethereum":
			address = generateEthereumAddress(job.seed)
		case "bitcoin":
			address = generateBitcoinAddress(job.seed)
		case "solana":
			address = generateSolanaAddress(job.seed)
		}

		results <- Result{index: job.index, address: address}
	}
}

func generateEthereumAddress(seed string) string {
	// Convert seed to private key
	seedBytes, err := hex.DecodeString(seed)
	if err != nil {
		log.Fatal("Invalid seed:", err)
	}

	// Create private key from seed
	privateKey, err := crypto.ToECDSA(seedBytes)
	if err != nil {
		log.Fatal("Failed to create private key:", err)
	}

	// Get Ethereum address
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	return address.Hex()
}

func generateBitcoinAddress(seed string) string {
	// Convert seed to private key
	seedBytes, err := hex.DecodeString(seed)
	if err != nil {
		log.Fatal("Invalid seed:", err)
	}

	// Create private key from seed
	privKey, _ := btcec.PrivKeyFromBytes(seedBytes)

	// Get Bitcoin address
	wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		log.Fatal("Failed to create WIF:", err)
	}

	addressPubKey, err := btcutil.NewAddressPubKey(wif.SerializePubKey(), &chaincfg.MainNetParams)
	if err != nil {
		log.Fatal("Failed to create address:", err)
	}

	return addressPubKey.EncodeAddress()
}

func generateSolanaAddress(seed string) string {
	// Convert seed to private key
	seedBytes, err := hex.DecodeString(seed)
	if err != nil {
		log.Fatal("Invalid seed:", err)
	}

	// Use seed bytes as private key
	account, err := types.AccountFromSeed(seedBytes)
	if err != nil {
		log.Fatal("Failed to create Solana account:", err)
	}
	return account.PublicKey.ToBase58()
}
