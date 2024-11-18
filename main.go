package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type FileContent struct {
	URI     string `json:"uri"`
	Content string `json:"content"`
}

type TranspileRequest struct {
	Content []FileContent `json:"content"`
	Output  string        `json:"output"`
}

type TranspileBlock struct {
	ID          string    `json:"id"`
	Provider    string    `json:"provider"`
	Provisioner string    `json:"provisioner"`
	Language    string    `json:"language"`
	Code        string    `json:"code"`
	CreatedAt   time.Time `json:"created_at"`
}

type TranspileResponse struct {
	Result struct {
		Blocks []TranspileBlock `json:"blocks"`
	} `json:"result"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: tf-eraser <directory>")
		os.Exit(1)
	}

	dirPath := os.Args[1]
	files := []FileContent{}
	// Read directory entries (non-recursive)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("error reading directory: %v", err)
		os.Exit(1)
	}

	// Process only .tf files in the current directory
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Check if file has .tf extension
		path := filepath.Join(dirPath, entry.Name())
		if filepath.Ext(path) == ".tf" {
			// Read file content
			content, err := os.ReadFile(path)
			if err != nil {
				fmt.Printf("error reading file %s: %v", path, err)
				os.Exit(1)
			}

			// Convert path to URI format
			absPath, err := filepath.Abs(path)
			if err != nil {
				fmt.Printf("error getting absolute path for %s: %v", path, err)
				os.Exit(1)

			}
			fileURI := "file://" + filepath.ToSlash(absPath)

			files = append(files, FileContent{
				URI:     fileURI,
				Content: string(content),
			})
		}
	}

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}

	// Prepare request
	reqBody := TranspileRequest{
		Content: files,
		Output:  "EraserDSL",
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Make HTTP request
	resp, err := http.Post("http://localhost:4000/v1/commands/Terraform/transpile",
		"application/json",
		bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read and print response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	var response TranspileResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing response JSON: %v\n", err)
		os.Exit(1)
	}

	// Print parsed blocks
	for _, block := range response.Result.Blocks {
		fmt.Println(block.Code)
	}
}
