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
	// resp, err := http.Post("http://localhost:4000/v1/commands/Terraform/transpile",
	resp, err := http.Post("https://apiv2.stakpak.dev/v1/commands/Terraform/transpile",
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
		// Generate Eraser diagram if API key is available
		if eraserAPIKey := os.Getenv("ERASER_API_KEY"); eraserAPIKey != "" {
			fmt.Println("\nFound Eraser API key, generating diagram image URL...")
			// Prepare request to Eraser API
			eraserURL := "https://app.eraser.io/api/render/elements"
			eraserReqBody := map[string]interface{}{
				"elements": []map[string]interface{}{
					{
						"type":        "diagram",
						"diagramType": "cloud-architecture-diagram",
						"code":        block.Code,
					},
				},
			}
			eraserJSON, err := json.Marshal(eraserReqBody)
			if err != nil {
				fmt.Printf("Error preparing Eraser request: %v\n", err)
				continue
			}

			// Create request
			eraserReq, err := http.NewRequest("POST", eraserURL, bytes.NewBuffer(eraserJSON))
			if err != nil {
				fmt.Printf("Error creating Eraser request: %v\n", err)
				continue
			}

			// Add headers
			eraserReq.Header.Add("accept", "application/json")
			eraserReq.Header.Add("content-type", "application/json")
			eraserReq.Header.Add("authorization", "Bearer "+eraserAPIKey)

			// Make request
			eraserResp, err := http.DefaultClient.Do(eraserReq)
			if err != nil {
				fmt.Printf("Error making Eraser request: %v\n", err)
				continue
			}
			defer eraserResp.Body.Close()

			// Read response
			eraserBody, err := io.ReadAll(eraserResp.Body)
			if err != nil {
				fmt.Printf("Error reading Eraser response: %v\n", err)
				continue
			}

			// Parse response
			var eraserResponse struct {
				ImageURL            string `json:"imageUrl"`
				CreateEraserFileURL string `json:"createEraserFileUrl"`
			}
			if err := json.Unmarshal(eraserBody, &eraserResponse); err != nil {
				fmt.Printf("Error parsing Eraser response: %v\n", err)
				continue
			}

			fmt.Printf("Eraser Diagram Image URL: %s\n", eraserResponse.ImageURL)
		}
	}
}
