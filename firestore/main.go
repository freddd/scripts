package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func performDatabaseGet(url string, token string) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("❌ Failed to create request: %v", err)
		return
	}

	req.Header.Set("Accept", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("An error occurred with the network request: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response body: %v", err)
		return
	}

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Println("✅ Request successful (200 OK). Database metadata retrieved.")
		var prettyJSON any
		if err := json.Unmarshal(body, &prettyJSON); err != nil {
			log.Printf("❌ Failed to parse JSON response: %v", err)
			return
		}
		prettyBody, err := json.MarshalIndent(prettyJSON, "", "    ")
		if err != nil {
			log.Printf("❌ Failed to format JSON: %v", err)
			return
		}
		fmt.Println(string(prettyBody))
	case http.StatusUnauthorized, http.StatusForbidden:
		fmt.Printf("❌ Request failed (%d Permission Denied).\n", resp.StatusCode)
		if token == "" {
			fmt.Println("   This is the expected result for an unauthenticated user, as database metadata is not public.")
		} else {
			fmt.Println("   The provided token may be invalid, expired, or lack the 'firestore.databases.get' permission.")
		}
	case http.StatusNotFound:
		fmt.Printf("ℹ️  Request failed (404 Not Found). The project or database does not exist.\n")
	default:
		fmt.Printf("❌ An error occurred. Status Code: %d\n", resp.StatusCode)
		fmt.Printf("   Response: %s\n", string(body))
	}
}

func main() {
	apiKey := flag.String("api-key", "", "Google Cloud API key. (Required)")
	projectID := flag.String("project-id", "", "Your Google Cloud project ID. (Required)")
	databaseID := flag.String("database-id", "(default)", "The Firestore database ID (usually '(default)').")
	token := flag.String("token", "", "Optional. An OAuth 2.0 bearer token for authenticated requests.")

	flag.Parse()

	if *apiKey == "" || *projectID == "" {
		fmt.Println("❌ Missing required flags: --api-key and --project-id are required.")
		flag.Usage()
		os.Exit(1)
	}

	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/%s?key=%s", *projectID, *databaseID, *apiKey)

	fmt.Printf("▶️  Targeting Firestore Database: '%s' in project '%s'\n", *databaseID, *projectID)
	fmt.Printf("   API Endpoint: %s\n", url)

	fmt.Println("\n" + "==================== UNAUTHENTICATED GET =====================")
	performDatabaseGet(url, "")

	fmt.Println("\n" + "==================== AUTHENTICATED GET =======================")
	if *token != "" {
		performDatabaseGet(url, *token)
	} else {
		fmt.Println("ℹ️  No --token provided. Skipping authenticated check.")
	}
}
