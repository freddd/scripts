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

type FirestoreDocument struct {
	Name       string         `json:"name"`
	Fields     map[string]any `json:"fields"`
	CreateTime string         `json:"createTime"`
	UpdateTime string         `json:"updateTime"`
}

func parseFirestoreFields(fields map[string]any) map[string]any {
	parsedData := make(map[string]any)
	if fields == nil {
		return parsedData
	}

	for key, value := range fields {
		valueDict, ok := value.(map[string]any)
		if !ok {
			parsedData[key] = value
			continue
		}

		for valueType, actualValue := range valueDict {
			if valueType == "mapValue" {
				mapValue, ok := actualValue.(map[string]any)
				if ok {
					nestedFields, ok := mapValue["fields"].(map[string]any)
					if ok {
						parsedData[key] = parseFirestoreFields(nestedFields)
					}
				}
			} else {
				parsedData[key] = actualValue
			}
			break
		}
	}
	return parsedData
}

func performFirestoreGet(url string, token string) {
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
		fmt.Println("✅ Request successful (200 OK). Document found and is readable.")
		var doc FirestoreDocument
		if err := json.Unmarshal(body, &doc); err != nil {
			log.Printf("❌ Failed to parse JSON response: %v", err)
			return
		}

		fmt.Printf("   - Name: %s\n", doc.Name)
		fmt.Printf("   - Create Time: %s\n", doc.CreateTime)
		fmt.Printf("   - Update Time: %s\n", doc.UpdateTime)

		fmt.Println("   - Fields (parsed):")
		parsedFields := parseFirestoreFields(doc.Fields)
		prettyFields, err := json.MarshalIndent(parsedFields, "", "    ")
		if err != nil {
			log.Printf("❌ Failed to format parsed fields as JSON: %v", err)
			return
		}
		fmt.Println(string(prettyFields))

	case http.StatusNotFound:
		fmt.Println("ℹ️  Request failed (404 Not Found). The document does not exist at this path.")
	case http.StatusUnauthorized, http.StatusForbidden:
		fmt.Printf("❌ Request failed (%d Permission Denied).\n", resp.StatusCode)
		if token == "" {
			fmt.Println("   This is the expected result for an unauthenticated user on a protected database.")
		} else {
			fmt.Println("   The provided token may be invalid, expired, or lack the required permissions.")
		}
	default:
		fmt.Printf("❌ An error occurred. Status Code: %d\n", resp.StatusCode)
		fmt.Printf("   Response: %s\n", string(body))
	}
}

func main() {
	apiKey := flag.String("api-key", "", "Google Cloud API key. (Required)")
	projectID := flag.String("project-id", "", "Your Google Cloud project ID. (Required)")
	documentPath := flag.String("document-path", "", "The full path to the document (e.g., 'users/user123'). (Required)")
	databaseID := flag.String("database-id", "(default)", "The Firestore database ID (usually '(default)').")
	token := flag.String("token", "", "Optional. An OAuth 2.0 bearer token for authenticated requests.")

	flag.Parse()

	if *apiKey == "" || *projectID == "" || *documentPath == "" {
		fmt.Println("❌ Missing required flags: --api-key, --project-id, and --document-path are required.")
		flag.Usage()
		os.Exit(1)
	}

	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/%s/documents/%s?key=%s", *projectID, *databaseID, *documentPath, *apiKey)

	fmt.Printf("▶️  Targeting Firestore document: '%s'\n", *documentPath)
	fmt.Printf("   API Endpoint: %s\n", url)

	fmt.Println("\n" + "==================== UNAUTHENTICATED GET =====================")
	performFirestoreGet(url, "")

	fmt.Println("\n" + "===================== AUTHENTICATED GET ======================")
	if *token != "" {
		performFirestoreGet(url, *token)
	} else {
		fmt.Println("ℹ️  No --token provided. Skipping authenticated check.")
	}
}
