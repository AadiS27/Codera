package sandbox

import (
	"errors"
	"time"
)

// Profile represents the resource limits and configuration for a sandbox execution.
type Profile struct {
	Name             string
	Image            string
	MemoryLimitBytes int64
	CPULimit         float64 // e.g., 0.5 for half a core, 1.0 for full core
	PidsLimit        int64
	TmpfsSizeBytes   int64
	Timeout          time.Duration
	MaxOutputBytes   int64
}

var (
	ErrUnsupportedLanguage = errors.New("unsupported language")
)

// Default profiles
var (
	ProfileSmall = Profile{
		Name:             "SMALL",
		MemoryLimitBytes: 128 * 1024 * 1024, // 128 MB
		CPULimit:         0.5,
		PidsLimit:        128,
		TmpfsSizeBytes:   32 * 1024 * 1024,  // 32 MB
		Timeout:          2 * time.Second,
		MaxOutputBytes:   1024 * 1024,       // 1 MB
	}

	ProfileMedium = Profile{
		Name:             "MEDIUM",
		MemoryLimitBytes: 256 * 1024 * 1024, // 256 MB
		CPULimit:         1.0,
		PidsLimit:        256,
		TmpfsSizeBytes:   64 * 1024 * 1024,
		Timeout:          15 * time.Second,
		MaxOutputBytes:   5 * 1024 * 1024,
	}

	ProfileLarge = Profile{
		Name:             "LARGE",
		MemoryLimitBytes: 512 * 1024 * 1024, // 512 MB
		CPULimit:         2.0,
		PidsLimit:        512,
		TmpfsSizeBytes:   128 * 1024 * 1024,
		Timeout:          10 * time.Second,
		MaxOutputBytes:   10 * 1024 * 1024,
	}
)

// GetProfileForLanguage maps a language to its approved sandbox profile and image.
// In production, these images would be custom minimal images per language.
// For now, we use standard images with minimal tags.
func GetProfileForLanguage(lang string) (Profile, error) {
	// For testing Phase 8, we map all to ProfileMedium by default,
	// but assign language-specific images.
	profile := ProfileMedium

	switch lang {
	case "python":
		profile.Image = "python:3.11-alpine" // Ideally codera-sandbox-python
	case "java":
		profile.Image = "eclipse-temurin:21-jdk-alpine"
	case "go":
		profile.Image = "golang:1.22-alpine"
	case "c", "cpp":
		profile.Image = "gcc:13" // Alpine gcc lacks some common libs out of the box, use debian-based for now or alpine if careful
	case "sql":
		profile.Image = "keinos/sqlite3"
	default:
		return Profile{}, ErrUnsupportedLanguage
	}

	return profile, nil
}
