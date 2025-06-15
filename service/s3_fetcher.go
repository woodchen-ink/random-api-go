package service

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"random-api-go/model"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Fetcher S3获取器
type S3Fetcher struct {
	timeout time.Duration
}

// NewS3Fetcher 创建S3获取器
func NewS3Fetcher() *S3Fetcher {
	return &S3Fetcher{
		timeout: 30 * time.Second,
	}
}

// FetchURLs 从S3存储桶获取文件URL列表
func (sf *S3Fetcher) FetchURLs(s3Config *model.S3Config) ([]string, error) {
	if s3Config == nil {
		return nil, fmt.Errorf("S3配置不能为空")
	}

	// 验证必需的配置
	if s3Config.Endpoint == "" {
		return nil, fmt.Errorf("S3端点地址不能为空")
	}
	if s3Config.BucketName == "" {
		return nil, fmt.Errorf("存储桶名称不能为空")
	}
	if s3Config.AccessKeyID == "" {
		return nil, fmt.Errorf("访问密钥ID不能为空")
	}
	if s3Config.SecretAccessKey == "" {
		return nil, fmt.Errorf("访问密钥不能为空")
	}

	// 创建S3客户端
	client, err := sf.createS3Client(s3Config)
	if err != nil {
		return nil, fmt.Errorf("创建S3客户端失败: %w", err)
	}

	// 获取对象列表
	objects, err := sf.listObjects(client, s3Config)
	if err != nil {
		return nil, fmt.Errorf("获取对象列表失败: %w", err)
	}

	// 过滤和转换为URL
	urls := sf.convertObjectsToURLs(objects, s3Config)

	log.Printf("从S3存储桶 %s 获取到 %d 个文件URL", s3Config.BucketName, len(urls))
	return urls, nil
}

// createS3Client 创建S3客户端
func (sf *S3Fetcher) createS3Client(s3Config *model.S3Config) (*s3.Client, error) {
	// 设置默认地区
	region := s3Config.Region
	if region == "" {
		region = "us-east-1"
	}

	// 创建凭证
	creds := credentials.NewStaticCredentialsProvider(
		s3Config.AccessKeyID,
		s3Config.SecretAccessKey,
		"",
	)

	// 创建配置
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("加载AWS配置失败: %w", err)
	}

	// 创建S3客户端选项
	options := func(o *s3.Options) {
		if s3Config.Endpoint != "" {
			o.BaseEndpoint = aws.String(s3Config.Endpoint)
		}
		o.UsePathStyle = s3Config.UsePathStyle
	}

	client := s3.NewFromConfig(cfg, options)
	return client, nil
}

// listObjects 列出存储桶中的对象
func (sf *S3Fetcher) listObjects(client *s3.Client, s3Config *model.S3Config) ([]types.Object, error) {
	ctx, cancel := context.WithTimeout(context.Background(), sf.timeout)
	defer cancel()

	var allObjects []types.Object
	var continuationToken *string

	// 设置前缀（文件夹路径）
	prefix := strings.TrimPrefix(s3Config.FolderPath, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// 设置分隔符（如果不包含子文件夹）
	var delimiter *string
	if !s3Config.IncludeSubfolders {
		delimiter = aws.String("/")
	}

	// 确定使用的ListObjects版本
	listVersion := s3Config.ListObjectsVersion
	if listVersion == "" {
		listVersion = "v2" // 默认使用v2
	}

	for {
		if listVersion == "v1" {
			// 使用ListObjects (v1)
			input := &s3.ListObjectsInput{
				Bucket:    aws.String(s3Config.BucketName),
				Prefix:    aws.String(prefix),
				Delimiter: delimiter,
				MaxKeys:   aws.Int32(1000),
			}

			if continuationToken != nil {
				input.Marker = continuationToken
			}

			result, err := client.ListObjects(ctx, input)
			if err != nil {
				return nil, fmt.Errorf("ListObjects失败: %w", err)
			}

			allObjects = append(allObjects, result.Contents...)

			if !aws.ToBool(result.IsTruncated) {
				break
			}

			if len(result.Contents) > 0 {
				continuationToken = result.Contents[len(result.Contents)-1].Key
			}
		} else {
			// 使用ListObjectsV2 (v2)
			input := &s3.ListObjectsV2Input{
				Bucket:            aws.String(s3Config.BucketName),
				Prefix:            aws.String(prefix),
				Delimiter:         delimiter,
				MaxKeys:           aws.Int32(1000),
				ContinuationToken: continuationToken,
			}

			result, err := client.ListObjectsV2(ctx, input)
			if err != nil {
				return nil, fmt.Errorf("ListObjectsV2失败: %w", err)
			}

			allObjects = append(allObjects, result.Contents...)

			if !aws.ToBool(result.IsTruncated) {
				break
			}

			continuationToken = result.NextContinuationToken
		}
	}

	return allObjects, nil
}

// convertObjectsToURLs 将S3对象转换为URL列表
func (sf *S3Fetcher) convertObjectsToURLs(objects []types.Object, s3Config *model.S3Config) []string {
	var urls []string

	// 编译文件扩展名正则表达式
	var extensionRegexes []*regexp.Regexp
	for _, ext := range s3Config.FileExtensions {
		if ext != "" {
			// 确保扩展名以点开头
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			// 转义特殊字符并创建正则表达式
			pattern := regexp.QuoteMeta(ext) + "$"
			if regex, err := regexp.Compile("(?i)" + pattern); err == nil {
				extensionRegexes = append(extensionRegexes, regex)
			} else {
				log.Printf("警告: 无效的文件扩展名正则表达式 '%s': %v", ext, err)
			}
		}
	}

	for _, obj := range objects {
		if obj.Key == nil {
			continue
		}

		key := aws.ToString(obj.Key)

		// 跳过以/结尾的对象（文件夹）
		if strings.HasSuffix(key, "/") {
			continue
		}

		// 如果设置了文件扩展名过滤，检查是否匹配
		if len(extensionRegexes) > 0 {
			matched := false
			for _, regex := range extensionRegexes {
				if regex.MatchString(key) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// 生成URL
		fileURL := sf.generateURL(key, s3Config)
		if fileURL != "" {
			urls = append(urls, fileURL)
		}
	}

	return urls
}

// generateURL 生成文件的访问URL
func (sf *S3Fetcher) generateURL(key string, s3Config *model.S3Config) string {
	// 如果设置了自定义域名
	if s3Config.CustomDomain != "" {
		return sf.generateCustomDomainURL(key, s3Config)
	}

	// 使用S3标准URL格式
	return sf.generateS3URL(key, s3Config)
}

// generateCustomDomainURL 生成自定义域名URL
func (sf *S3Fetcher) generateCustomDomainURL(key string, s3Config *model.S3Config) string {
	baseURL := strings.TrimSuffix(s3Config.CustomDomain, "/")

	// 处理key路径
	path := key
	if s3Config.RemoveBucket {
		// 如果需要移除bucket名称，并且key以bucket名称开头
		bucketPrefix := s3Config.BucketName + "/"
		if strings.HasPrefix(path, bucketPrefix) {
			path = strings.TrimPrefix(path, bucketPrefix)
		}
	}

	// 对路径进行适当的URL编码，但保留路径分隔符
	path = sf.encodeURLPath(path)

	// 确保路径以/开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return baseURL + path
}

// generateS3URL 生成S3标准URL
func (sf *S3Fetcher) generateS3URL(key string, s3Config *model.S3Config) string {
	// 对key进行适当的URL编码，但保留路径分隔符
	encodedKey := sf.encodeURLPath(key)

	if s3Config.UsePathStyle {
		// Path-style URL: http://endpoint/bucket/key
		endpoint := strings.TrimSuffix(s3Config.Endpoint, "/")
		return fmt.Sprintf("%s/%s/%s", endpoint, s3Config.BucketName, encodedKey)
	} else {
		// Virtual-hosted-style URL: http://bucket.endpoint/key
		endpoint := strings.TrimSuffix(s3Config.Endpoint, "/")

		// 解析endpoint以获取主机名
		if parsedURL, err := url.Parse(endpoint); err == nil {
			scheme := parsedURL.Scheme
			if scheme == "" {
				scheme = "https"
			}
			host := parsedURL.Host
			if host == "" {
				host = parsedURL.Path
			}
			return fmt.Sprintf("%s://%s.%s/%s", scheme, s3Config.BucketName, host, encodedKey)
		}

		// 如果解析失败，回退到path-style
		return fmt.Sprintf("%s/%s/%s", endpoint, s3Config.BucketName, encodedKey)
	}
}

// encodeURLPath 对URL路径进行编码，保留路径分隔符，但编码其他特殊字符
func (sf *S3Fetcher) encodeURLPath(path string) string {
	// 分割路径为各个部分
	parts := strings.Split(path, "/")

	// 对每个部分进行URL编码
	for i, part := range parts {
		if part != "" {
			// 使用PathEscape对每个路径段进行编码
			// 这会将空格编码为%20，这是URL路径中的标准做法
			parts[i] = url.PathEscape(part)
		}
	}

	// 重新组合路径
	return strings.Join(parts, "/")
}
