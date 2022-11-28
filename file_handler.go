package main

import (
	"bytes"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	maxContentSize    int64 = 1024 * 1024 * 1000 // 100 MB
	minUploadPartSize int64 = 1024 * 1024 * 5    // 5 MB
)

type Link struct {
	URL string `json:"url"`
}

type Message struct {
	Key   string `json:"key"`
	Links []Link `json:"links"`
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		contentType := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "video/") {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		if r.ContentLength > maxContentSize {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		ctx := r.Context()
		multipartUploadOutput, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
			Bucket:                    aws.String(bucket),
			Key:                       aws.String(uuid.New().String()),
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
		var buffer bytes.Buffer
		var completedParts []types.CompletedPart
		var lastPart bool
		var partNumber int32 = 1 // The first part number must always start with 1.
		for !lastPart {
			n, err := io.CopyN(&buffer, r.Body, minUploadPartSize)
			// The io.EOF error occurs when the stream has reached its end.
			if n == 0 || err == io.EOF {
				lastPart = true
			} else if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// If the buffer has the minimum required size or the current part is the last one,
			// a new part is stored in the bucket.
			uploadPartOutput, err := client.UploadPart(ctx, &s3.UploadPartInput{
				Bucket:               multipartUploadOutput.Bucket,
				Key:                  multipartUploadOutput.Key,
				PartNumber:           partNumber,
				UploadId:             multipartUploadOutput.UploadId,
				Body:                 bytes.NewReader(buffer.Bytes()),
				ContentLength:        int64(buffer.Len()),
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
			// The buffer is empty to the next parts.
			buffer.Reset()
			partNumber++
		}
		completeMultipartUploadOutput, err := client.CompleteMultipartUpload(ctx,
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(Message{
			Key: *completeMultipartUploadOutput.Key,
			Links: []Link{
				{
					URL: *completeMultipartUploadOutput.Location,
				},
			},
		}); err != nil {
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
