package assets

import (
	"embed"
	"fmt"
	"log"
	"sync"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
)

//go:embed audio
var AudioFS embed.FS

// PredecodedAudio holds a predecoded audio stream and its format
type PredecodedAudio struct {
	Buffer beep.Buffer
	Format beep.Format
}

// PredecodedCache holds all predecoded audio files
type PredecodedCache struct {
	mu    sync.RWMutex
	cache map[string]*PredecodedAudio
}

var (
	predecodedCache *PredecodedCache
	initOnce        sync.Once
)

// GetPredecodedCache returns the singleton predecoded cache
func GetPredecodedCache() *PredecodedCache {
	initOnce.Do(func() {
		predecodedCache = &PredecodedCache{
			cache: make(map[string]*PredecodedAudio),
		}
		predecodedCache.loadAllAudio()
	})
	return predecodedCache
}

// loadAllAudio preloads and decodes all MP3 files in the audio directory
func (pc *PredecodedCache) loadAllAudio() {
	log.Println("Preloading and decoding audio files...")

	entries, err := AudioFS.ReadDir("audio")
	if err != nil {
		log.Fatalf("Failed to read audio directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := fmt.Sprintf("audio/%s", entry.Name())
		if err := pc.preloadFile(filePath); err != nil {
			log.Printf("Failed to preload %s: %v", filePath, err)
		} else {
			log.Printf("Successfully preloaded %s", filePath)
		}
	}

	log.Printf("Preloading complete. Loaded %d audio files.", len(pc.cache))
}

// preloadFile loads and decodes a single MP3 file into the cache
func (pc *PredecodedCache) preloadFile(filePath string) error {
	file, err := AudioFS.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	streamer, format, err := mp3.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode MP3 %s: %w", filePath, err)
	}

	// Convert streamer to buffer to store in memory
	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	streamer.Close()

	pc.mu.Lock()
	pc.cache[filePath] = &PredecodedAudio{
		Buffer: *buffer,
		Format: format,
	}
	pc.mu.Unlock()

	return nil
}

// GetAudio retrieves a predecoded audio file
func (pc *PredecodedCache) GetAudio(filePath string) (*PredecodedAudio, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	audio, exists := pc.cache[filePath]
	return audio, exists
}
