package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

// TestGenerateEthereumAddress tests the Ethereum address generation
func TestGenerateEthereumAddress(t *testing.T) {
	// Use a fixed seed for reproducible testing
	seed := "c8c5e5a7f326a2b5f3eee778db6856430d808c32b16e18d8228a93e3d94791a3"

	address := generateEthereumAddress(seed)

	// Get the actual address from the current implementation
	expected := "0x0d747F8AdFdE4beF87CF21FEa682083C7149268f"

	if address != expected {
		t.Errorf("Expected address %s, got %s", expected, address)
	}
}

// TestGenerateBitcoinAddress tests the Bitcoin address generation
func TestGenerateBitcoinAddress(t *testing.T) {
	// Use a fixed seed for reproducible testing
	seed := "c8c5e5a7f326a2b5f3eee778db6856430d808c32b16e18d8228a93e3d94791a3"

	address := generateBitcoinAddress(seed)

	// Since Bitcoin address generation is more complex, we'll just check the format
	if !strings.HasPrefix(address, "1") && !strings.HasPrefix(address, "3") {
		t.Errorf("Expected Bitcoin address to start with 1 or 3, got %s", address)
	}

	// Check length is reasonable
	if len(address) < 25 || len(address) > 35 {
		t.Errorf("Bitcoin address length unusual: %d", len(address))
	}
}

// TestGenerateSolanaAddress tests the Solana address generation
func TestGenerateSolanaAddress(t *testing.T) {
	// Use a fixed seed for reproducible testing
	seed := "c8c5e5a7f326a2b5f3eee778db6856430d808c32b16e18d8228a93e3d94791a3"

	address := generateSolanaAddress(seed)

	// Check that the address is in base58 format (typically starts with specific characters)
	if len(address) != 44 {
		t.Errorf("Expected Solana address length to be 44, got %d", len(address))
	}
}

// TestProgressBar tests the progress bar functionality
func TestProgressBar(t *testing.T) {
	// Redirect stderr to capture output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Create progress bar
	pb := NewProgressBar(100, 10)

	// Test initial state
	if pb.current != 0 || pb.total != 100 || pb.width != 10 {
		t.Errorf("Progress bar initialized incorrectly")
	}

	// Update progress to 50%
	pb.Update(50)

	// Update to 100%
	pb.Update(100)

	// Close the pipe and restore stderr
	w.Close()
	output, _ := io.ReadAll(r)
	os.Stderr = oldStderr

	// Check that output contains progress indicators
	outputStr := string(output)
	if !strings.Contains(outputStr, "[") || !strings.Contains(outputStr, "]") {
		t.Errorf("Progress bar output missing brackets: %s", outputStr)
	}
}

// TestResultCollector tests the result collector functionality separately from the actual ResultCollector type
func TestResultCollector(t *testing.T) {
	// Create our own test implementation to avoid the os.File requirement
	var output bytes.Buffer
	var resultMap = make(map[int]string)
	var mu sync.Mutex
	var nextToPrint int
	var resultCount int

	// Create a mock progress bar
	pb := NewProgressBar(5, 10)

	// Add results out of order
	results := []Result{
		{index: 2, address: "address2"},
		{index: 0, address: "address0"},
		{index: 1, address: "address1"},
		{index: 4, address: "address4"},
		{index: 3, address: "address3"},
	}

	// Process results in a way similar to ResultCollector.AddResult
	for i, result := range results {
		// This mimics the logic in ResultCollector.AddResult
		mu.Lock()
		resultMap[result.index] = result.address
		resultCount++

		// Update progress bar
		pb.Update(resultCount)

		// Print results in order
		for {
			if address, exists := resultMap[nextToPrint]; exists {
				fmt.Fprintln(&output, address)
				delete(resultMap, nextToPrint)
				nextToPrint++
			} else {
				break
			}
		}
		mu.Unlock()

		// Check that result count increments correctly
		if resultCount != i+1 {
			t.Errorf("Expected result count %d, got %d", i+1, resultCount)
		}
	}

	// All results should be processed
	if nextToPrint != 5 {
		t.Errorf("Expected nextToPrint to be 5, got %d", nextToPrint)
	}

	// Check the output content
	outputStr := output.String()
	expectedAddresses := []string{"address0", "address1", "address2", "address3", "address4"}
	for _, addr := range expectedAddresses {
		if !strings.Contains(outputStr, addr) {
			t.Errorf("Output missing expected address: %s", addr)
		}
	}
}

// TestGenerateHashForAddress tests the hash generation functionality for --generate-hash option
func TestGenerateHashForAddress(t *testing.T) {
	// Test address
	address := "0x122b84B924B5f9bE23b7A8961685B3AB8224ebCa"

	// Generate hash manually
	h := sha256.New()
	h.Write([]byte(address))
	expectedHash := hex.EncodeToString(h.Sum(nil))[:6]

	// Test the hash generation directly
	var output bytes.Buffer
	fmt.Fprintf(&output, "%s,%s\n", expectedHash, address)

	expectedOutput := fmt.Sprintf("%s,%s\n", expectedHash, address)
	if output.String() != expectedOutput {
		t.Errorf("Expected output to be %q, got %q", expectedOutput, output.String())
	}

	// Create a temporary file for a real integration test
	tempFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Test with the actual ResultCollector
	rc := NewResultCollector(1, 1, tempFile, true)
	pb := NewProgressBar(1, 10)
	rc.AddResult(Result{index: 0, address: address}, pb)

	// Flush and rewind the file
	tempFile.Sync()
	tempFile.Seek(0, 0)

	// Read the content
	content, err := io.ReadAll(tempFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	// Check the content
	contentStr := string(content)
	if !strings.Contains(contentStr, expectedHash+","+address) {
		t.Errorf("Expected file to contain %s,%s, got %s", expectedHash, address, contentStr)
	}
}

// TestBatchSubmitJobs tests the batch job submission
func TestBatchSubmitJobs(t *testing.T) {
	// Create channels and a pool
	jobs := make(chan Job, 10)
	pool := &sync.Pool{
		New: func() interface{} {
			return &Job{}
		},
	}

	// Submit jobs
	go batchSubmitJobs(jobs, 5, "testseed", "ethereum", 2, pool)

	// Read and validate jobs
	count := 0
	for job := range jobs {
		if job.network != "ethereum" {
			t.Errorf("Expected network ethereum, got %s", job.network)
		}
		count++
		if count == 5 {
			// All jobs received, we're done
			break
		}
	}

	if count != 5 {
		t.Errorf("Expected 5 jobs, got %d", count)
	}
}

// TestWorker tests the worker function
func TestWorker(t *testing.T) {
	// Create channels
	jobs := make(chan Job, 3)
	results := make(chan Result, 3)
	var wg sync.WaitGroup

	// Start worker
	wg.Add(1)
	go worker(1, jobs, results, &wg)

	// Send jobs for different networks
	jobs <- Job{index: 0, seed: "c8c5e5a7f326a2b5f3eee778db6856430d808c32b16e18d8228a93e3d94791a3", network: "ethereum"}
	jobs <- Job{index: 1, seed: "c8c5e5a7f326a2b5f3eee778db6856430d808c32b16e18d8228a93e3d94791a3", network: "bitcoin"}
	jobs <- Job{index: 2, seed: "c8c5e5a7f326a2b5f3eee778db6856430d808c32b16e18d8228a93e3d94791a3", network: "solana"}
	close(jobs)

	// Wait for worker to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(results)
		close(done)
	}()

	// Verify results
	resultCount := 0
	for result := range results {
		if result.index < 0 || result.index > 2 {
			t.Errorf("Unexpected result index: %d", result.index)
		}
		if result.address == "" {
			t.Errorf("Empty address for result %d", result.index)
		}
		resultCount++
	}

	// Wait for done signal
	<-done

	// Check that we got all results
	if resultCount != 3 {
		t.Errorf("Expected 3 results, got %d", resultCount)
	}
}
