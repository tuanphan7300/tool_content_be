package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"google.golang.org/api/option"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

// VoiceSample đại diện cho một voice sample đã được cache
type VoiceSample struct {
	VoiceName    string    `json:"voice_name"`
	LanguageCode string    `json:"language_code"`
	DisplayName  string    `json:"display_name"`
	Gender       string    `json:"gender"`
	Quality      string    `json:"quality"`
	SampleText   string    `json:"sample_text"`
	AudioPath    string    `json:"audio_path"`
	AudioURL     string    `json:"audio_url"`
	FileSize     int64     `json:"file_size"`
	CreatedAt    time.Time `json:"created_at"`
}

// VoiceCacheService quản lý cache voice samples
type VoiceCacheService struct {
	samplesDir string
	client     *texttospeech.Client
	mutex      sync.RWMutex
	samples    map[string]*VoiceSample // voice_name -> VoiceSample
}

var (
	voiceCacheService *VoiceCacheService
	voiceCacheMutex   sync.Mutex
)

// InitVoiceCacheService khởi tạo voice cache service
func InitVoiceCacheService() (*VoiceCacheService, error) {
	voiceCacheMutex.Lock()
	defer voiceCacheMutex.Unlock()

	if voiceCacheService != nil {
		return voiceCacheService, nil
	}

	// Tạo thư mục samples
	samplesDir := "./storage/voice_samples"
	if err := os.MkdirAll(samplesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create samples directory: %v", err)
	}

	// Khởi tạo Google TTS client
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile("data/google_clound_tts_api.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS client: %v", err)
	}

	voiceCacheService = &VoiceCacheService{
		samplesDir: samplesDir,
		client:     client,
		samples:    make(map[string]*VoiceSample),
	}

	// Generate tất cả voice samples
	go voiceCacheService.GenerateAllVoiceSamples()

	return voiceCacheService, nil
}

// GetVoiceCacheService trả về instance của voice cache service
func GetVoiceCacheService() *VoiceCacheService {
	if voiceCacheService == nil {
		service, err := InitVoiceCacheService()
		if err != nil {
			log.Printf("Failed to initialize voice cache service: %v", err)
			return nil
		}
		return service
	}
	return voiceCacheService
}

// GenerateAllVoiceSamples tạo tất cả voice samples
func (vcs *VoiceCacheService) GenerateAllVoiceSamples() {
	log.Println("🎵 [VOICE CACHE] Bắt đầu generate tất cả voice samples...")

	voices := GetAvailableVoices()
	totalVoices := 0
	generatedCount := 0

	// Đếm tổng số voices
	for _, languageVoices := range voices {
		totalVoices += len(languageVoices)
	}

	// Generate từng voice
	for language, languageVoices := range voices {
		for _, voice := range languageVoices {
			if err := vcs.GenerateVoiceSample(voice, language); err != nil {
				log.Printf("❌ [VOICE CACHE] Failed to generate sample for %s: %v", voice.Name, err)
			} else {
				generatedCount++
				log.Printf("✅ [VOICE CACHE] Generated sample %d/%d: %s", generatedCount, totalVoices, voice.Name)
			}
		}
	}

	log.Printf("🎵 [VOICE CACHE] Hoàn thành! Đã generate %d/%d voice samples", generatedCount, totalVoices)
}

// GenerateVoiceSample tạo sample cho một voice cụ thể
func (vcs *VoiceCacheService) GenerateVoiceSample(voice VoiceOption, language string) error {
	// Kiểm tra xem sample đã tồn tại chưa
	samplePath := filepath.Join(vcs.samplesDir, fmt.Sprintf("%s_sample.mp3", voice.Name))
	if _, err := os.Stat(samplePath); err == nil {
		// Sample đã tồn tại, load vào cache
		return vcs.LoadVoiceSample(voice, language, samplePath)
	}

	// Tạo sample text dựa trên ngôn ngữ
	sampleText := vcs.GetSampleText(language)

	// Tạo TTS request
	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: sampleText},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: voice.LanguageCode,
			Name:         voice.Name,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   texttospeechpb.AudioEncoding_MP3,
			SpeakingRate:    1.0,
			SampleRateHertz: 44100,
		},
	}

	// Gọi Google TTS API
	ctx := context.Background()
	resp, err := vcs.client.SynthesizeSpeech(ctx, req)
	if err != nil {
		return fmt.Errorf("TTS API call failed: %v", err)
	}

	// Lưu audio file
	if err := os.WriteFile(samplePath, resp.AudioContent, 0644); err != nil {
		return fmt.Errorf("failed to save sample file: %v", err)
	}

	// Load sample vào cache
	return vcs.LoadVoiceSample(voice, language, samplePath)
}

// LoadVoiceSample load voice sample vào cache
func (vcs *VoiceCacheService) LoadVoiceSample(voice VoiceOption, language string, samplePath string) error {
	// Lấy file info
	fileInfo, err := os.Stat(samplePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Tạo VoiceSample
	sample := &VoiceSample{
		VoiceName:    voice.Name,
		LanguageCode: voice.LanguageCode,
		DisplayName:  voice.DisplayName,
		Gender:       voice.Gender,
		Quality:      voice.Quality,
		SampleText:   vcs.GetSampleText(language),
		AudioPath:    samplePath,
		AudioURL:     fmt.Sprintf("http://localhost:8888/voice-samples/%s_sample.mp3", voice.Name),
		FileSize:     fileInfo.Size(),
		CreatedAt:    fileInfo.ModTime(),
	}

	// Thêm vào cache
	vcs.mutex.Lock()
	vcs.samples[voice.Name] = sample
	vcs.mutex.Unlock()

	return nil
}

// GetVoiceSample trả về voice sample theo voice name
func (vcs *VoiceCacheService) GetVoiceSample(voiceName string) *VoiceSample {
	vcs.mutex.RLock()
	defer vcs.mutex.RUnlock()

	if sample, exists := vcs.samples[voiceName]; exists {
		// Trả về full URL thay vì relative path
		if !strings.HasPrefix(sample.AudioURL, "http") {
			sample.AudioURL = "http://localhost:8888/voice-samples/" + filepath.Base(sample.AudioURL)
		}
		return sample
	}
	return nil
}

// GetAllVoiceSamples trả về tất cả voice samples
func (vcs *VoiceCacheService) GetAllVoiceSamples() []*VoiceSample {
	vcs.mutex.RLock()
	defer vcs.mutex.RUnlock()

	var samples []*VoiceSample
	for _, sample := range vcs.samples {
		// Trả về full URL thay vì relative path
		if !strings.HasPrefix(sample.AudioURL, "http") {
			sample.AudioURL = "http://localhost:8888/voice-samples/" + filepath.Base(sample.AudioURL)
		}
		samples = append(samples, sample)
	}
	return samples
}

// GetSampleText trả về sample text phù hợp cho từng ngôn ngữ
func (vcs *VoiceCacheService) GetSampleText(language string) string {
	sampleTexts := map[string]string{
		"vi": "Xin chào! Đây là giọng đọc mẫu để bạn có thể nghe thử và chọn giọng phù hợp nhất.",
		"en": "Hello! This is a sample voice for you to preview and choose the most suitable one.",
	}

	if text, exists := sampleTexts[language]; exists {
		return text
	}

	// Default to English
	return sampleTexts["en"]
}

// RefreshVoiceSamples refresh tất cả voice samples
func (vcs *VoiceCacheService) RefreshVoiceSamples() error {
	log.Println("🔄 [VOICE CACHE] Refreshing all voice samples...")

	// Xóa tất cả samples cũ
	vcs.mutex.Lock()
	vcs.samples = make(map[string]*VoiceSample)
	vcs.mutex.Unlock()

	// Generate lại
	go vcs.GenerateAllVoiceSamples()

	return nil
}
