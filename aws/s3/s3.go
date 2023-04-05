package s3

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/Invicton-Labs/go-common/aws/credentials"
	"github.com/Invicton-Labs/go-common/gensync"
	"github.com/Invicton-Labs/go-common/log"
	"github.com/Invicton-Labs/go-stackerr"
	awsarn "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/smithy-go/transport/http"
)

var s3Clients gensync.Map[string, *s3.Client]
var s3ClientInitOnces gensync.Map[string, gensync.Once]

func getS3ClientRegion(ctx context.Context, region string) (*s3.Client, stackerr.Error) {
	once, _ := s3ClientInitOnces.LoadOrStore(region, gensync.Once{})
	if err := once.Do(func() stackerr.Error {
		creds, err := credentials.GetCredentialsProvider(ctx)
		if err != nil {
			return err
		}
		s3Client := s3.New(s3.Options{
			Region:      region,
			Credentials: creds,
			Logger:      log.GetAwsLogger(),
		})
		s3Clients.Store(region, s3Client)
		return nil
	}); err != nil {
		return nil, err
	}

	client, _ := s3Clients.Load(region)
	return client, nil
}

var s3Client *s3.Client
var s3ClientInitOnce gensync.Once

func getS3Client(ctx context.Context) (*s3.Client, stackerr.Error) {
	if err := s3ClientInitOnce.Do(func() stackerr.Error {
		// config, err := credentials.GetConfig(ctx)
		// if err != nil {
		// 	return err
		// }
		creds, err := credentials.GetCredentialsProvider(ctx)
		if err != nil {
			return err
		}
		s3Client = s3.New(s3.Options{
			Region:      "us-east-1",
			Credentials: creds,
			Logger:      log.GetAwsLogger(),
		})
		return nil
	}); err != nil {
		return nil, err
	}

	return s3Client, nil
}

type PutObjectArgs struct {
	ContentEncoding    *string
	ContentType        *string
	ContentLanguage    *string
	ContentDisposition *string
}

func PutObject[ContentType string | *string | []byte | *bytes.Reader | *strings.Reader | *gzip.Reader | *bytes.Buffer](ctx context.Context, arn string, content ContentType, args *PutObjectArgs) stackerr.Error {
	parsedArn, cerr := awsarn.Parse(arn)
	if cerr != nil {
		return stackerr.Wrap(cerr)
	}
	client, err := getS3Client(ctx)
	if err != nil {
		return err
	}

	parts := strings.SplitN(parsedArn.Resource, "/", 1)
	bucket := parts[0]
	key := parts[1]

	uploader := manager.NewUploader(client)

	var bodyInterface interface{} = content

	var bodyReader io.Reader
	if r, ok := bodyInterface.(io.Reader); ok {
		bodyReader = r
	} else {
		switch v := bodyInterface.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case *string:
			bodyReader = strings.NewReader(*v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		case *bytes.Reader:
			bodyReader = v
		case *strings.Reader:
			bodyReader = v
		default:
			return stackerr.Errorf("Unknown content variable type: %T", content)
		}
	}

	input := &s3.PutObjectInput{
		Bucket:            aws.String(bucket),
		Key:               aws.String(key),
		Body:              bodyReader,
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
		BucketKeyEnabled:  true,
	}

	if args != nil {
		input.ContentType = args.ContentType
		input.ContentEncoding = args.ContentEncoding
		input.ContentLanguage = args.ContentLanguage
		input.ContentDisposition = args.ContentDisposition
	}

	_, cerr = uploader.Upload(ctx, input)
	if cerr != nil {
		return stackerr.Wrap(cerr)
	}
	return nil
}

func GetObject(ctx context.Context, arn string, disableChecksumVerification ...bool) ([]byte, stackerr.Error) {
	parsedArn, cerr := awsarn.Parse(arn)
	if cerr != nil {
		return nil, stackerr.Wrap(cerr)
	}
	client, err := getS3Client(ctx)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(parsedArn.Resource, "/", 2)
	bucket := parts[0]
	key := parts[1]

	// Do a HEAD request to find out how many bytes the file is
	head, cerr := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if cerr != nil {
		return nil, stackerr.Wrap(cerr)
	}

	// Create a buffer of the correct length
	buffer := manager.NewWriteAtBuffer(make([]byte, 0, head.ContentLength))

	downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.Logger = log.GetAwsLogger()
	})

	checksumMode := types.ChecksumModeEnabled
	if len(disableChecksumVerification) > 0 && disableChecksumVerification[0] {
		checksumMode = ""
	}

	// Try to get the specific version of the file, so the content length
	// remains the same.
	if _, err := downloader.Download(ctx, buffer, &s3.GetObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		ChecksumMode: checksumMode,
		VersionId:    head.VersionId,
	}); err != nil {
		var re *http.ResponseError
		if errors.As(err, &re) {
			if re.HTTPStatusCode() == 403 {
				// If access was forbidden, try it again without specifying the version ID
				if _, err = downloader.Download(ctx, buffer, &s3.GetObjectInput{
					Bucket:       aws.String(bucket),
					Key:          aws.String(key),
					ChecksumMode: checksumMode,
				}); err != nil {
					return nil, stackerr.Wrap(err)
				}
				log.Debugf("Could not download a specific version of s3://%s/%s, used fallback to downloading current version", bucket, key)
			}
		} else {
			return nil, stackerr.Wrap(err)
		}
	}
	return buffer.Bytes(), nil
}
