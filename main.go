package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func getenv_default(key, default_value string) string {
	value := os.Getenv(key)
	if value == "" {
		return default_value
	}
	return value
}

// S3 configuration
var (
	Region    = getenv_default("AWS_REGION", "us-east-1")
	Bucket    = getenv_default("S3_BUCKET", "my-bucket")  // The S3 bucket to upload to
	S3KeyPath = getenv_default("S3_KEY_PATH", "uploads/") // The S3 key path
)

// S3 client initialization
var s3Client *s3.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(Region))
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %s", err)
	}

	s3Client = s3.NewFromConfig(cfg)
}

func uploadToS3(filename string, file io.ReadSeeker) error {
	_, err := s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(Bucket),
		Key:    aws.String(S3KeyPath + filename),
		Body:   file,
		ACL:    types.ObjectCannedACLPrivate, // You can set ACL to "public-read" or others
	})

	if err != nil {
		return fmt.Errorf("failed to upload to s3: %v", err)
	}

	return nil
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form data
	err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get file and file name from form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Open a temporary file to store the uploaded file contents
	tempFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		http.Error(w, "Error creating temp file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	// Copy file contents to the temporary file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// Reopen the temporary file for reading
	tempFile.Seek(0, io.SeekStart)

	// Upload the file to S3
	err = uploadToS3(header.Filename, tempFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error uploading file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully!"))
}

func listFilesHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := s3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(Bucket),
		Prefix: aws.String(S3KeyPath),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing files: %v", err), http.StatusInternalServerError)
		return
	}

	var files []string
	for _, item := range resp.Contents {
		files = append(files, *item.Key)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Files in S3: %v", files)))
}

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/upload", uploadHandler)

	http.HandleFunc("/list", listFilesHandler)

	port := getenv_default("PORT", "8080")
	fmt.Printf("Server is running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
