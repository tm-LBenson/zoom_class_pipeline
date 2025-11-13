package main

import (
	"context"
	"encoding/json"
	"fmt"
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

type Config struct {
	WatchDir           string `json:"watchDir"`
	Bucket             string `json:"bucket"`
	Region             string `json:"region"`
	Prefix             string `json:"prefix"`
	BaseURL            string `json:"baseUrl"`
	JSONPath           string `json:"jsonPath"`
	TopicPrefix        string `json:"topicPrefix"`
	AWSAccessKeyID     string `json:"awsAccessKeyId"`
	AWSSecretAccessKey string `json:"awsSecretAccessKey"`
}

type Recording struct {
	ID       string `json:"id"`
	Topic    string `json:"topic"`
	Start    string `json:"start"`
	Duration string `json:"duration"`
	Link     string `json:"link"`
	File     string `json:"file"`
}

func defaultConfig() Config {
	return Config{
		WatchDir:           "/path/to/zoom/recordings",
		Bucket:             "codex-recordings",
		Region:             "us-east-1",
		Prefix:             "level1",
		BaseURL:            "",
		JSONPath:           "./recordings.json",
		TopicPrefix:        "Level 1",
		AWSAccessKeyID:     "YOUR_ACCESS_KEY_ID",
		AWSSecretAccessKey: "YOUR_SECRET_ACCESS_KEY",
	}
}

func loadConfig() Config {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.json"
	}
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
		log.Fatalf("config file created at %s, fill in values and run again", path)
	}
	if err != nil {
		log.Fatal(err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatal(err)
	}
	if cfg.WatchDir == "" || cfg.Bucket == "" || cfg.Region == "" || cfg.JSONPath == "" {
		log.Fatal("watchDir, bucket, region, jsonPath are required in config.json")
	}
	if cfg.TopicPrefix == "" {
		cfg.TopicPrefix = "Class"
	}
	if cfg.AWSAccessKeyID != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", cfg.AWSAccessKeyID)
	}
	if cfg.AWSSecretAccessKey != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", cfg.AWSSecretAccessKey)
	}
	return cfg
}

func loadRecordings(path string) []Recording {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte("[]\n"), 0644); err != nil {
			log.Fatal(err)
		}
		return []Recording{}
	}
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

func saveRecordings(path string, items []Recording) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}
	}
	tmp := path + ".tmp"
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		log.Fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Fatal(err)
	}
}

func newS3Client(region string) *s3.Client {
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
	}
	return s3.NewFromConfig(awsCfg)
}

func buildBaseURL(cfg Config) string {
	if cfg.BaseURL != "" {
		return strings.TrimRight(cfg.BaseURL, "/")
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
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

func isStableFile(info os.FileInfo) bool {
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

func makeKey(cfg Config, path string, info os.FileInfo) string {
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
	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix != "" {
		return fmt.Sprintf("%s/%s/%s", prefix, date, base)
	}
	return fmt.Sprintf("%s/%s", date, base)
}

func uploadFile(ctx context.Context, client *s3.Client, cfg Config, path string) (string, error) {
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
	link := fmt.Sprintf("%s/%s", baseURL, key)
	return link, nil
}

func addRecording(items []Recording, cfg Config, filePath string, link string) []Recording {
	info, err := os.Stat(filePath)
	if err != nil {
		log.Println("stat error after upload", err)
		return items
	}
	start := info.ModTime().UTC().Format(time.RFC3339)
	fileName := filepath.Base(filePath)
	dateLabel := info.ModTime().Format("2006-01-02")
	topic := fmt.Sprintf("%s %s", cfg.TopicPrefix, dateLabel)
	r := Recording{
		ID:       fileName,
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
	items := loadRecordings(cfg.JSONPath)
	known := existingFiles(items)
	files := listNewFiles(cfg.WatchDir, known)
	if len(files) == 0 {
		log.Println("no new files")
		return
	}
	client := newS3Client(cfg.Region)
	ctx := context.Background()
	updated := items
	for _, path := range files {
		log.Println("uploading", path)
		link, err := uploadFile(ctx, client, cfg, path)
		if err != nil {
			log.Println("upload failed", err)
			continue
		}
		updated = addRecording(updated, cfg, path, link)
	}
	saveRecordings(cfg.JSONPath, updated)
	log.Println("updated recordings.json")
}
