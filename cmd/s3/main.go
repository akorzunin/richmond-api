package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"richmond-api/internal/s3"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: .env file not found")
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(
			os.Stderr,
			"  upload <file path> <object key>  Upload file to S3\n",
		)
		fmt.Fprintf(
			os.Stderr,
			"  download <object key> <file path>  Download file from S3\n",
		)
		fmt.Fprintf(
			os.Stderr,
			"  list-buckets                    List all buckets\n",
		)
		fmt.Fprintf(
			os.Stderr,
			"  list-objects                    List objects in main bucket\n",
		)
		fmt.Fprintf(
			os.Stderr,
			"  create-bucket <bucket name>     Create a new bucket\n",
		)
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  S3_ENDPOINT   (required)\n")
		fmt.Fprintf(os.Stderr, "  S3_ACCESS_KEY (required)\n")
		fmt.Fprintf(os.Stderr, "  S3_SECRET_KEY (required)\n")
		fmt.Fprintf(os.Stderr, "  S3_USE_SSL    (default: false)\n")
		fmt.Fprintf(os.Stderr, "  S3_BUCKET     (default: main)\n")
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := s3.NewClientFromEnv()
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}
	client := cfg.Client

	cmd := flag.Arg(0)
	switch cmd {
	case "upload":
		if flag.NArg() != 3 {
			fmt.Fprintf(
				os.Stderr,
				"Usage: %s upload <file path> <object key>\n",
				os.Args[0],
			)
			os.Exit(1)
		}
		filePath := flag.Arg(1)
		objectKey := flag.Arg(2)
		err = s3.UploadCommand(client, cfg.Bucket, filePath, objectKey)
		if err != nil {
			log.Fatalf("Upload failed: %v", err)
		}
		fmt.Printf(
			"Uploaded %s to s3://%s/%s\n",
			filePath,
			cfg.Bucket,
			objectKey,
		)

	case "download":
		if flag.NArg() != 3 {
			fmt.Fprintf(
				os.Stderr,
				"Usage: %s download <object key> <file path>\n",
				os.Args[0],
			)
			os.Exit(1)
		}
		objectKey := flag.Arg(1)
		filePath := flag.Arg(2)
		err = s3.DownloadCommand(client, cfg.Bucket, objectKey, filePath)
		if err != nil {
			log.Fatalf("Download failed: %v", err)
		}
		fmt.Printf(
			"Downloaded s3://%s/%s to %s\n",
			cfg.Bucket,
			objectKey,
			filePath,
		)

	case "list-buckets":
		err = s3.ListBucketsCommand(client)
		if err != nil {
			log.Fatalf("List buckets failed: %v", err)
		}

	case "list-objects":
		err = s3.ListObjectsCommand(client, cfg.Bucket)
		if err != nil {
			log.Fatalf("List objects failed: %v", err)
		}

	case "create-bucket":
		if flag.NArg() != 2 {
			fmt.Fprintf(
				os.Stderr,
				"Usage: %s create-bucket <bucket name>\n",
				os.Args[0],
			)
			os.Exit(1)
		}
		bucketName := flag.Arg(1)
		err = s3.EnsureBucketExists(client, bucketName)
		if err != nil {
			log.Fatalf("Create bucket failed: %v", err)
		}
		fmt.Printf("Created bucket %s\n", bucketName)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}
