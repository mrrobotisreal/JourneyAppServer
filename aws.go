package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"net/http"
	"time"
)

var s3Client *s3.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(fmt.Errorf("unable to load AWS config: %w", err))
	}

	s3Client = s3.NewFromConfig(cfg)
}

func GeneratePresignedURL(bucket, key string) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)

	req, err := presignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func PresignHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	uuid := r.URL.Query().Get("uuid")
	filename := r.URL.Query().Get("filename")

	if username == "" || uuid == "" || filename == "" {
		http.Error(w, "Missing query params", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s/%s/%s", username, uuid, filename)
	url, err := GeneratePresignedURL("my-journey-app", key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"url": %q}`, url)
}
