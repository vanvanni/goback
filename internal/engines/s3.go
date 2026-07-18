package engines

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/vanvanni/goback/internal/helper"
)

type S3Config struct {
	Bucket            string `toml:"bucket" yaml:"bucket"`
	Region            string `toml:"region" yaml:"region"`
	AccessKey         string `toml:"access_key" yaml:"access_key"`
	SecretKey         string `toml:"secret_key" yaml:"secret_key"`
	Endpoint          string `toml:"endpoint" yaml:"endpoint"`
	ForcePathStyle    bool   `toml:"force-path" yaml:"force-path"`
	HostnameImmutable bool   `toml:"hostname-immutable" yaml:"hostname-immutable"`
}

type S3 struct {
	s3Client *s3.Client
	bucket   string
}

func (s *S3) Kind() EngineKind {
	return EngineKindS3
}

func NewClient(ctx context.Context, s3Config S3Config) (*S3, error) {
	var opts []func(*config.LoadOptions) error

	if s3Config.Region != "" {
		opts = append(opts, config.WithRegion(s3Config.Region))
	}

	if s3Config.Endpoint != "" {
		opts = append(opts, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               s3Config.Endpoint,
						SigningRegion:     s3Config.Region,
						HostnameImmutable: s3Config.HostnameImmutable,
					}, nil
				},
			),
		))
	}

	if s3Config.AccessKey != "" && s3Config.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				s3Config.AccessKey,
				s3Config.SecretKey,
				"",
			),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if s3Config.ForcePathStyle {
			o.UsePathStyle = true
		}
	})

	return &S3{
		s3Client: s3Client,
		bucket:   s3Config.Bucket,
	}, nil
}

func (c *S3) UploadFile(ctx context.Context, s string, d string) error {
	f, err := helper.OpenReadOnlyFile(s)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	fileName := filepath.Base(s)
	key := filepath.Join(d, fileName)

	_, err = c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

func (c *S3) DownloadFile(ctx context.Context, remoteKey string, localPath string) error {
	out, err := helper.OpenWriteOnlyFile(localPath, 0600)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	result, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(remoteKey),
	})
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer result.Body.Close()

	if _, err := io.Copy(out, result.Body); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	return nil
}

func (c *S3) KeepMax(ctx context.Context, d string, max int) error {
	if max < 0 {
		return fmt.Errorf("max must be non-negative")
	}

	prefix := d
	if prefix != "" && prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}

	var objects []types.Object
	paginator := s3.NewListObjectsV2Paginator(c.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}
		objects = append(objects, page.Contents...)
	}

	if len(objects) <= max {
		return nil
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.After(*objects[j].LastModified)
	})

	toDelete := objects[max:]
	for _, obj := range toDelete {
		_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    obj.Key,
		})
		if err != nil {
			return fmt.Errorf("failed to delete object %s: %w", *obj.Key, err)
		}
	}

	return nil
}
