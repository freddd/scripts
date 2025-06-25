package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func createXLSXFromDir(dirPath, xlsxPath string) error {
	dirInfo, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("❌ Error: The directory was not found at '%s'", dirPath)
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("❌ Error: The path '%s' is not a directory", dirPath)
	}

	if _, err := os.Stat(filepath.Join(dirPath, "[Content_Types].xml")); os.IsNotExist(err) {
		fmt.Printf("⚠️  Warning: Directory '%s' might not be an unzipped XLSX file.\n", dirPath)
		fmt.Println("   (Could not find '[Content_Types].xml'). Proceeding anyway...")
	}

	fmt.Printf("▶️  Rebuilding XLSX from directory: '%s'\n", dirPath)

	xlsxFile, err := os.Create(xlsxPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer xlsxFile.Close()

	zipWriter := zip.NewWriter(xlsxFile)
	defer zipWriter.Close()

	mimetypePath := filepath.Join(dirPath, "mimetype")
	if _, err := os.Stat(mimetypePath); !os.IsNotExist(err) {
		fmt.Println("   - Adding 'mimetype' (uncompressed)...")
		writer, err := zipWriter.CreateHeader(&zip.FileHeader{
			Name:   "mimetype",
			Method: zip.Store,
		})
		if err != nil {
			return fmt.Errorf("failed to create mimetype header in zip: %w", err)
		}
		mimetypeFile, err := os.Open(mimetypePath)
		if err != nil {
			return fmt.Errorf("failed to open mimetype file: %w", err)
		}
		defer mimetypeFile.Close()
		if _, err := io.Copy(writer, mimetypeFile); err != nil {
			return fmt.Errorf("failed to copy mimetype content: %w", err)
		}
	} else {
		fmt.Println("   - Warning: 'mimetype' file not found. The output XLSX may be invalid.")
	}

	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "mimetype" {
			return nil
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		fmt.Printf("   - Adding '%s'...\n", relPath)

		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return fmt.Errorf("failed to create entry for %s in zip: %w", relPath, err)
		}

		fileToZip, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer fileToZip.Close()

		_, err = io.Copy(writer, fileToZip)
		return err
	})

	if err != nil {
		return fmt.Errorf("error walking the path %s: %w", dirPath, err)
	}

	fmt.Printf("\n✅ Success! Rebuilt XLSX file saved to '%s'\n", filepath.Base(xlsxPath))
	return nil
}

func main() {
	inputDir := flag.String("in", "", "The path to the input directory (the unzipped XLSX contents). (Required)")
	outputXlsx := flag.String("out", "", "The path for the final, rebuilt .xlsx file. (Required)")
	flag.Parse()

	if *inputDir == "" || *outputXlsx == "" {
		fmt.Println("❌ Missing required flags: -in and -out are required.")
		flag.Usage()
		os.Exit(1)
	}

	if err := createXLSXFromDir(*inputDir, *outputXlsx); err != nil {
		log.Fatalf("\n❌ An unexpected error occurred: %v", err)
	}
}
