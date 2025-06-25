package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2/google"
)

var PERMISSIONS_TO_CHECK = []string{
	"cloudfunctions.functions.call",
	"cloudfunctions.functions.invoke",
	"cloudfunctions.functions.delete",
	"cloudfunctions.functions.get",
	"cloudfunctions.functions.update",
	"cloudfunctions.functions.sourceCodeGet",
	"cloudfunctions.functions.sourceCodeSet",
	"cloudfunctions.functions.getIamPolicy",
	"cloudfunctions.functions.setIamPolicy",
	"cloudfunctions.operations.get",
	"cloudfunctions.operations.list",
}

type permissionsRequest struct {
	Permissions []string `json:"permissions"`
}

type permissionsResponse struct {
	Permissions []string `json:"permissions"`
}

func performPermissionCheck(url string, payload []byte, token string) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("❌ Failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
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

	if resp.StatusCode == http.StatusOK {
		var respJSON permissionsResponse
		if err := json.Unmarshal(body, &respJSON); err != nil {
			log.Printf("❌ Failed to parse JSON response: %v", err)
			return
		}

		grantedPermissions := make(map[string]bool)
		for _, p := range respJSON.Permissions {
			grantedPermissions[p] = true
		}

		fmt.Println("✅ = Granted, ❌ = Not Granted\n")
		for _, permission := range PERMISSIONS_TO_CHECK {
			if grantedPermissions[permission] {
				fmt.Printf("✅ %s\n", permission)
			} else {
				fmt.Printf("❌ %s\n", permission)
			}
		}
		fmt.Println("------------------------------------------------------------")
		fmt.Printf("Found %d granted permissions out of %d checked.\n", len(grantedPermissions), len(PERMISSIONS_TO_CHECK))
	} else {
		fmt.Printf("ℹ️  Request failed with Status Code: %d\n", resp.StatusCode)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			fmt.Println("   This is expected for unauthenticated users if the function is not public.")
		} else {
			fmt.Printf("   Response: %s\n", string(body))
		}
	}
}

func main() {
	projectID := flag.String("project-id", "", "Your Google Cloud project ID. (Required)")
	location := flag.String("location", "", "The location/region of the Cloud Function (e.g., us-central1). (Required)")
	functionName := flag.String("function-name", "", "The name of the Cloud Function. (Required)")

	flag.Parse()

	if *projectID == "" || *location == "" || *functionName == "" {
		fmt.Println("❌ Missing required flags: --project-id, --location, and --function-name are required.")
		flag.Usage()
		os.Exit(1)
	}

	resource := fmt.Sprintf("projects/%s/locations/%s/functions/%s", *projectID, *location, *functionName)
	url := fmt.Sprintf("https://cloudfunctions.googleapis.com/v1/%s:testIamPermissions", resource)

	reqPayload := permissionsRequest{Permissions: PERMISSIONS_TO_CHECK}
	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		log.Fatalf("❌ Failed to create request JSON: %v", err)
	}

	fmt.Printf("▶️  Targeting function: '%s' in project '%s'...\n", *functionName, *projectID)

	fmt.Println("\n" + "==================== UNAUTHENTICATED CHECK ====================")
	performPermissionCheck(url, payloadBytes, "")

	fmt.Println("\n" + "===================== AUTHENTICATED CHECK =====================")
	ctx := context.Background()
	tokenSource, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		fmt.Println("❌ Error: Could not get authentication credentials for the authenticated check.")
		fmt.Println("   Please run 'gcloud auth application-default login'.")
		fmt.Printf("   Details: %v\n", err)
		return
	}

	token, err := tokenSource.Token()
	if err != nil {
		fmt.Printf("❌ Error: Could not retrieve token from credentials: %v\n", err)
		return
	}
	performPermissionCheck(url, payloadBytes, token.AccessToken)
}
