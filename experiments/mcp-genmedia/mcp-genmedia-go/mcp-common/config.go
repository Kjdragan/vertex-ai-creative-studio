package common

import (
	"log"
	"os"
)

type Config struct {
	ProjectID         string
	Location          string
	GenmediaBucket    string
	ImagenBucketPath  string
	VeoBucketPath     string
	Chirp3BucketPath  string
	LyriaBucketPath   string
	AvtoolBucketPath  string
	ApiEndpoint       string // New field
}

func LoadConfig() *Config {
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		log.Fatal("PROJECT_ID environment variable not set. Please set the env variable, e.g. export PROJECT_ID=$(gcloud config get project)")
	}

	return &Config{
		ProjectID:         projectID,
		Location:          GetEnv("LOCATION", "us-central1"),
		GenmediaBucket:    GetEnv("GENMEDIA_BUCKET", ""),
		ImagenBucketPath:  GetEnv("IMAGEN_BUCKET_PATH", ""),
		VeoBucketPath:     GetEnv("VEO_BUCKET_PATH", ""),
		Chirp3BucketPath:  GetEnv("CHIRP3_BUCKET_PATH", ""),
		LyriaBucketPath:   GetEnv("LYRIA_BUCKET_PATH", ""),
		AvtoolBucketPath:  GetEnv("AVTOOL_BUCKET_PATH", ""),
		ApiEndpoint:       os.Getenv("VERTEX_API_ENDPOINT"), // Use os.Getenv for optional value
	}
}

// GetEnv retrieves an environment variable by its key.
// If the variable is not set or is empty, it returns a fallback value.
// This function is useful for providing default values for optional configurations.
func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	if fallback != "" {
	    log.Printf("Environment variable %s not set or empty, using fallback: %s", key, fallback)
	} else {
	    log.Printf("Environment variable %s not set or empty, using empty fallback.", key)
	}
	return fallback
}
