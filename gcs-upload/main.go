package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type APIRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Status      string `json:"status"`
	Folder      string `json:"folder,omitempty"`
	ACL         string `json:"acl,omitempty"`
}

type APIResponse struct {
	URL string `json:"url"`
}

func getUploadURL(apiURL, filename, folder, acl, authToken string) (string, error) {
	fmt.Printf("▶️  Step 1: Requesting upload URL from '%s'...\n", apiURL)

	payload := APIRequest{
		FileName:    filename,
		ContentType: "application/zip",
		Status:      "uploading",
		Folder:      folder,
		ACL:         acl,
	}

	if folder != "" {
		fmt.Printf("   - Target folder: %s\n", folder)
	}
	if acl != "" {
		fmt.Printf("   - ACL permissions: %s\n", acl)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to create request JSON: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create API request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned non-200 status code: %d. Response: %s", resp.StatusCode, string(body))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("error parsing API response: %w", err)
	}

	if apiResp.URL == "" {
		return "", fmt.Errorf("API response did not contain a 'url' field")
	}

	return apiResp.URL, nil
}

func uploadFile(uploadURL, filePath string) error {
	fmt.Printf("\n▶️  Step 2: Uploading '%s' to pre-signed URL...\n", filepath.Base(filePath))

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("the file was not found at '%s': %w", filePath, err)
	}
	defer file.Close()

	req, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "application/zip")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error during file upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with non-200 status code: %d. Response: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✅ File uploaded successfully!\n")
	fmt.Printf("   - Status Code: %d\n", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		fmt.Println("   - Server Response:")
		fmt.Println(string(body))
	}

	return nil
}

func main() {
	apiURL := flag.String("api-url", "", "The URL of your API that provides the pre-signed upload URL. (Required)")
	filePath := flag.String("file-path", "", "The local path to the ZIP file to upload. (Required)")
	folder := flag.String("folder", "", "Optional. The target folder/directory for the upload.")
	acl := flag.String("acl", "", "Optional. The access control list (ACL) permissions for the uploaded file.")
	token := flag.String("token", "", "Optional. An OAuth 2.0 bearer token for authenticating with your API.")
	flag.Parse()

	if *apiURL == "" || *filePath == "" {
		fmt.Println("❌ Missing required flags: --api-url and --file-path are required.")
		flag.Usage()
		os.Exit(1)
	}

	filename := filepath.Base(*filePath)
	uploadURL, err := getUploadURL(*apiURL, filename, *folder, *acl, *token)
	if err != nil {
		log.Fatalf("\nWorkflow failed at Step 1. Reason: %v", err)
	}

	if err := uploadFile(uploadURL, *filePath); err != nil {
		log.Fatalf("\nWorkflow failed at Step 2. Reason: %v", err)
	}
}
