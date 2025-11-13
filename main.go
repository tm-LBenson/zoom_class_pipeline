package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type AppConfig struct {
	WatchDir           string `json:"watchDir"`
	Bucket             string `json:"bucket"`
	Region             string `json:"region"`
	VideoPrefix        string `json:"videoPrefix"`
	BaseURL            string `json:"baseUrl"`
	TopicPrefix        string `json:"topicPrefix"`
	AWSAccessKeyID     string `json:"awsAccessKeyId"`
	AWSSecretAccessKey string `json:"awsSecretAccessKey"`
}

type Recording struct {
	ID       string `json:"id"`
	Level    string `json:"level"`
	Topic    string `json:"topic"`
	Start    string `json:"start"`
	Duration string `json:"duration"`
	Link     string `json:"link"`
	File     string `json:"file"`
}

const indexKey = "recordings.json"

func defaultConfig() AppConfig {
	return AppConfig{
		WatchDir:           "/path/to/zoom/recordings",
		Bucket:             "codex-recordings-yourname",
		Region:             "us-east-1",
		VideoPrefix:        "level1",
		BaseURL:            "",
		TopicPrefix:        "Level 1",
		AWSAccessKeyID:     "YOUR_ACCESS_KEY_ID",
		AWSSecretAccessKey: "YOUR_SECRET_ACCESS_KEY",
	}
}

func defaultConfigPath() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "config.json")
}

func loadConfig() AppConfig {
	path := defaultConfigPath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := defaultConfig()
		b, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0600); err != nil {
			log.Fatal(err)
		}
		fmt.Println("config.json created at", path)
		fmt.Println("Fill in your values and run this program again.")
		os.Exit(0)
	}
	if err != nil {
		log.Fatal(err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatal(err)
	}

	if cfg.WatchDir == "" || cfg.Bucket == "" || cfg.Region == "" {
		log.Fatal("watchDir, bucket, and region are required in config.json")
	}
	if cfg.TopicPrefix == "" {
		cfg.TopicPrefix = "Class"
	}
	if cfg.VideoPrefix == "" {
		cfg.VideoPrefix = "level1"
	}
	if cfg.AWSAccessKeyID != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", cfg.AWSAccessKeyID)
	}
	if cfg.AWSSecretAccessKey != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", cfg.AWSSecretAccessKey)
	}

	return cfg
}

func newS3Client(region string) *s3.Client {
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
	}
	return s3.NewFromConfig(awsCfg)
}

func buildBaseURL(cfg AppConfig) string {
	if cfg.BaseURL != "" {
		return strings.TrimRight(cfg.BaseURL, "/")
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
}

func loadRecordingsFromS3(ctx context.Context, client *s3.Client, cfg AppConfig) []Recording {
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(indexKey),
	})
	if err != nil {
		log.Println("no existing index at", indexKey, "starting with empty list")
		return []Recording{}
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		log.Fatal(err)
	}
	if len(data) == 0 {
		return []Recording{}
	}

	var items []Recording
	if err := json.Unmarshal(data, &items); err != nil {
		log.Fatal(err)
	}
	return items
}

func saveRecordingsToS3(ctx context.Context, client *s3.Client, cfg AppConfig, items []Recording) {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(indexKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		log.Fatal(err)
	}
}

func existingFiles(items []Recording) map[string]bool {
	m := make(map[string]bool)
	for _, r := range items {
		if r.File != "" {
			m[r.File] = true
		}
	}
	return m
}

func isStableFile(info fs.FileInfo) bool {
	if info.IsDir() {
		return false
	}
	if !strings.EqualFold(filepath.Ext(info.Name()), ".mp4") {
		return false
	}
	if time.Since(info.ModTime()) < 30*time.Second {
		return false
	}
	return true
}

func listNewFiles(root string, known map[string]bool) []string {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Println("walk error", err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.EqualFold(filepath.Ext(name), ".mp4") {
			return nil
		}
		if known[name] {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			log.Println("stat error", err)
			return nil
		}
		if !isStableFile(info) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return out
}

func makeKey(cfg AppConfig, path string, info fs.FileInfo) string {
	base := filepath.Base(path)
	parent := filepath.Base(filepath.Dir(path))
	parts := strings.Split(parent, " ")
	date := ""
	if len(parts) > 0 {
		date = parts[0]
	}
	if date == "" {
		date = info.ModTime().UTC().Format("2006-01-02")
	}
	prefix := strings.Trim(cfg.VideoPrefix, "/")
	if prefix != "" {
		return fmt.Sprintf("%s/%s/%s", prefix, date, base)
	}
	return fmt.Sprintf("%s/%s", date, base)
}

func uploadFile(ctx context.Context, client *s3.Client, cfg AppConfig, path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	key := makeKey(cfg, path, info)

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	if err != nil {
		return "", err
	}

	baseURL := buildBaseURL(cfg)
	return fmt.Sprintf("%s/%s", baseURL, key), nil
}

func addRecording(items []Recording, cfg AppConfig, filePath string, link string) []Recording {
	info, err := os.Stat(filePath)
	if err != nil {
		log.Println("stat error after upload", err)
		return items
	}
	start := info.ModTime().UTC().Format(time.RFC3339)
	fileName := filepath.Base(filePath)
	dateLabel := info.ModTime().Format("2006-01-02")
	level := cfg.TopicPrefix
	topic := fmt.Sprintf("%s %s", level, dateLabel)

	r := Recording{
		ID:       fileName,
		Level:    level,
		Topic:    topic,
		Start:    start,
		Duration: "",
		Link:     link,
		File:     fileName,
	}

	items = append(items, r)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Start > items[j].Start
	})
	return items
}

func main() {
	cfg := loadConfig()
	client := newS3Client(cfg.Region)
	ctx := context.Background()

	items := loadRecordingsFromS3(ctx, client, cfg)
	known := existingFiles(items)
	files := listNewFiles(cfg.WatchDir, known)
	updated := items

	if len(files) == 0 && len(updated) == 0 {
		saveRecordingsToS3(ctx, client, cfg, updated)
		log.Println("no recordings found in", cfg.WatchDir, "created empty index at", indexKey)
		return
	}

	if len(files) == 0 && len(updated) > 0 {
		log.Println("no new recordings in", cfg.WatchDir)
		return
	}

	for _, path := range files {
		log.Println("uploading", path)
		link, err := uploadFile(ctx, client, cfg, path)
		if err != nil {
			log.Println("upload failed", err)
			continue
		}
		updated = addRecording(updated, cfg, path, link)
	}

	saveRecordingsToS3(ctx, client, cfg, updated)
	log.Println("updated", indexKey)
}
