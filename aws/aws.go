package aws

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		panic(fmt.Errorf("unable to load AWS config: %w", err))
	}

	fmt.Println("Successfully loaded credentials...")
	s3Client = s3.NewFromConfig(cfg)
	fmt.Println("Successfully created new s3Client from config...")
}

func GeneratePresignedUploadURL(bucket, key string) (string, error) {
	fmt.Println("Generating a new presigned url...")
	presignClient := s3.NewPresignClient(s3Client)
	fmt.Println("Successfully created new presignClient")

	req, err := presignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func PresignPutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling request for PresignPutHandler...")
	username := r.URL.Query().Get("user")
	entryId := r.URL.Query().Get("entryId")
	filename := r.URL.Query().Get("filename")

	if username == "" || entryId == "" || filename == "" {
		http.Error(w, "Missing query params", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s/%s/%s/%s", "images", username, entryId, filename)
	fmt.Println("Key:", key)
	url, err := GeneratePresignedUploadURL("winapps-myjourney", key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("Url:", url)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"url": %q}`, url)
}

func generatePresignedGetURL(key string) (string, error) {
	bucket := "winapps-myjourney"
	presignClient := s3.NewPresignClient(s3Client)
	req, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(5*time.Hour))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func PresignGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing required param \"key\"", http.StatusBadRequest)
		return
	}

	url, err := generatePresignedGetURL(key)
	if err != nil {
		http.Error(w, "Error generating pre-signed GET URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"url": %q}`, url)
}
