package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"net/http"
	"os"
)

var (
	client *s3.Client
	bucket = os.Getenv("BUCKET")
)

func init() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client = s3.NewFromConfig(cfg)
}

func main() {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/api/v1/file", fileHandler)
	if err := http.ListenAndServe(":8081", serveMux); err != nil {
		log.Fatal(err)
	}
}
