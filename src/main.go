package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	maxContentSize    int64 = 1024 * 1024 * 2500 // 2500 MB
	minUploadPartSize int64 = 1024 * 1024 * 5    // 5 MB
)

var (
	s3Client *s3.Client
	bucket   = os.Getenv("BUCKET")
)

type Link struct {
	URL string `json:"url"`
}

type Message struct {
	Key   string `json:"key"`
	Links []Link `json:"links"`
}

func GetFilenameExtension(contentType string) string {
	switch contentType {
	case "image/gif":
		return ".gif"
	case "image/jpeg":
		return ".jpeg"
	case "image/png":
		return ".png"
	case "image/tiff":
		return ".tiff"
	case "video/quicktime":
		return ".mov"
	case "video/mpeg":
		return ".mpeg"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	default:
		return ""
	}
}

func FileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		w.Header().Set("Accept", "application/octet-stream")
		w.Header().Set("Content-Type", "application/json")
		contentType := r.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") && !strings.HasPrefix(contentType, "video/") {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		if r.ContentLength > maxContentSize {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		filenameExtension := GetFilenameExtension(contentType)
		ctx := r.Context()
		multipartUploadOutput, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
			Bucket:                    aws.String(bucket),
			Key:                       aws.String(uuid.New().String() + filenameExtension),
			ACL:                       types.ObjectCannedACLPrivate,
			BucketKeyEnabled:          false,
			CacheControl:              nil,
			ContentDisposition:        nil,
			ContentEncoding:           nil,
			ContentLanguage:           nil,
			ContentType:               aws.String(contentType),
			ExpectedBucketOwner:       nil,
			Expires:                   nil,
			GrantFullControl:          nil,
			GrantRead:                 nil,
			GrantReadACP:              nil,
			GrantWriteACP:             nil,
			Metadata:                  nil,
			ObjectLockLegalHoldStatus: "",
			ObjectLockMode:            "",
			ObjectLockRetainUntilDate: nil,
			RequestPayer:              "",
			SSECustomerAlgorithm:      nil,
			SSECustomerKey:            nil,
			SSECustomerKeyMD5:         nil,
			SSEKMSEncryptionContext:   nil,
			SSEKMSKeyId:               nil,
			ServerSideEncryption:      "",
			StorageClass:              "",
			Tagging:                   nil,
			WebsiteRedirectLocation:   nil,
		})
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Skip data already stored.
		/*
			if false {
				if _, err := io.CopyN(ioutil.Discard, r.Body, minUploadPartSize); err != nil {
					log.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		*/
		var buff bytes.Buffer
		var completedParts []types.CompletedPart
		var lastPart bool
		var partNumber int32 = 1 // The first part number must always start with 1.
		for !lastPart {
			n, err := io.CopyN(&buff, r.Body, minUploadPartSize)
			// The io.EOF error occurs when the stream has reached its end.
			if n == 0 || err == io.EOF {
				lastPart = true
			} else if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// If the buff has the minimum required size or the current part is the last one,
			// a new part is stored in the bucket.
			uploadPartOutput, err := s3Client.UploadPart(ctx, &s3.UploadPartInput{
				Bucket:               multipartUploadOutput.Bucket,
				Key:                  multipartUploadOutput.Key,
				PartNumber:           partNumber,
				UploadId:             multipartUploadOutput.UploadId,
				Body:                 bytes.NewReader(buff.Bytes()),
				ContentLength:        int64(buff.Len()),
				ContentMD5:           nil,
				ExpectedBucketOwner:  nil,
				RequestPayer:         "",
				SSECustomerAlgorithm: nil,
				SSECustomerKey:       nil,
				SSECustomerKeyMD5:    nil,
			})
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			completedParts = append(completedParts, types.CompletedPart{
				ETag:       uploadPartOutput.ETag,
				PartNumber: partNumber,
			})
			// The buff is empty to the next parts.
			buff.Reset()
			partNumber++
		}
		completeMultipartUploadOutput, err := s3Client.CompleteMultipartUpload(ctx,
			&s3.CompleteMultipartUploadInput{
				Bucket:              multipartUploadOutput.Bucket,
				Key:                 multipartUploadOutput.Key,
				UploadId:            multipartUploadOutput.UploadId,
				ExpectedBucketOwner: nil,
				MultipartUpload: &types.CompletedMultipartUpload{
					Parts: completedParts,
				},
				RequestPayer: "",
			})
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		b, err := json.Marshal(Message{
			Key: *completeMultipartUploadOutput.Key,
			Links: []Link{
				{
					URL: *completeMultipartUploadOutput.Location,
				},
			},
		})
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write(b); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func init() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	s3Client = s3.NewFromConfig(cfg)
}

func main() {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/api/v1/file", FileHandler)
	if err := http.ListenAndServe(":8080", serveMux); err != nil {
		log.Fatalln(err)
	}
}
