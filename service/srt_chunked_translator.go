package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// SRTChunkedTranslator xử lý translation SRT với chunking để tối ưu tốc độ
type SRTChunkedTranslator struct {
	maxChunkSize    int
	overlapSize     int
	maxConcurrent   int
	timeoutPerChunk time.Duration
	retryAttempts   int
}

// SRTChunk đại diện cho một phần nhỏ của file SRT
type SRTChunk struct {
	ChunkID    int
	StartIndex int
	EndIndex   int
	Content    string
	EntryCount int
	Processed  bool
	Result     string
	Error      error
	RetryCount int
}

// SRTChunkingStrategy chiến lược chia chunk
type SRTChunkingStrategy struct {
	MaxChunkSize    int           // Số câu tối đa mỗi chunk (mặc định: 50)
	OverlapSize     int           // Số câu overlap giữa các chunk (mặc định: 0)
	MaxConcurrent   int           // Số chunk xử lý đồng thời (mặc định: 5)
	TimeoutPerChunk time.Duration // Timeout cho mỗi chunk (mặc định: 60s)
	RetryAttempts   int           // Số lần retry (mặc định: 2)
}

// ChunkedTranslationResult kết quả translation với chunking
type ChunkedTranslationResult struct {
	TranslatedContent string
	ChunksProcessed   int
	TotalChunks       int
	ProcessingTime    time.Duration
	HasOverlap        bool
}

var (
	chunkedTranslator *SRTChunkedTranslator
	translatorMutex   sync.Mutex
)

// InitSRTChunkedTranslator khởi tạo SRT Chunked Translator
func InitSRTChunkedTranslator() *SRTChunkedTranslator {
	translatorMutex.Lock()
	defer translatorMutex.Unlock()

	if chunkedTranslator != nil {
		return chunkedTranslator
	}

	chunkedTranslator = &SRTChunkedTranslator{
		maxChunkSize:    50, // 50 câu mỗi chunk
		overlapSize:     0,  // 0 câu overlap
		maxConcurrent:   5,  // 5 chunks đồng thời
		timeoutPerChunk: 60 * time.Second,
		retryAttempts:   2,
	}

	log.Printf("SRT Chunked Translator initialized with %d max chunks, %d overlap, %d concurrent",
		chunkedTranslator.maxChunkSize, chunkedTranslator.overlapSize, chunkedTranslator.maxConcurrent)
	return chunkedTranslator
}

// GetSRTChunkedTranslator trả về instance của SRT Chunked Translator
func GetSRTChunkedTranslator() *SRTChunkedTranslator {
	if chunkedTranslator == nil {
		return InitSRTChunkedTranslator()
	}
	return chunkedTranslator
}

// TranslateSRTWithChunking dịch SRT với chunking (hàm mới)
// Hỗ trợ cả GPT và Gemini dựa trên service config
func (t *SRTChunkedTranslator) TranslateSRTWithChunking(
	srtFilePath, apiKey, modelName, targetLanguage string,
	strategy *SRTChunkingStrategy,
) (*ChunkedTranslationResult, error) {
	startTime := time.Now()
	log.Printf("🚀 [CHUNKED TRANSLATION] Bắt đầu chunked translation cho %s", srtFilePath)

	// Sử dụng strategy mặc định nếu không có
	if strategy == nil {
		strategy = &SRTChunkingStrategy{
			MaxChunkSize:    t.maxChunkSize,
			OverlapSize:     t.overlapSize,
			MaxConcurrent:   t.maxConcurrent,
			TimeoutPerChunk: t.timeoutPerChunk,
			RetryAttempts:   t.retryAttempts,
		}
	}

	// Đọc file SRT
	srtContent, err := os.ReadFile(srtFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SRT file: %v", err)
	}

	// Parse SRT content để đếm số câu
	entries, err := parseSRT(string(srtContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SRT: %v", err)
	}

	totalEntries := len(entries)
	log.Printf("📊 [CHUNKED TRANSLATION] Total SRT entries: %d", totalEntries)

	// Nếu ít câu hơn chunk size, sử dụng translation cũ
	if totalEntries <= strategy.MaxChunkSize {
		log.Printf("⚠️ [CHUNKED TRANSLATION] SRT chỉ có %d entries (≤ %d), chuyển sang TRADITIONAL translation", totalEntries, strategy.MaxChunkSize)

		// Tự động chọn service dựa trên modelName
		var translatedContent string
		var err error

		if strings.Contains(strings.ToLower(modelName), "gpt") {
			log.Printf("🔧 [TRADITIONAL] Sử dụng GPT API cho translation")
			translatedContent, err = TranslateSRTWithGPT(srtFilePath, apiKey, modelName, targetLanguage)
		} else {
			log.Printf("🔧 [TRADITIONAL] Sử dụng Gemini API cho translation")
			// Sử dụng Gemini
			translatedContent, err = TranslateSRTFileWithModelAndLanguage(srtFilePath, apiKey, modelName, targetLanguage)
		}

		if err != nil {
			return nil, err
		}

		log.Printf("✅ [TRADITIONAL] Translation hoàn thành trong %v", time.Since(startTime))
		return &ChunkedTranslationResult{
			TranslatedContent: translatedContent,
			ChunksProcessed:   1,
			TotalChunks:       1,
			ProcessingTime:    time.Since(startTime),
			HasOverlap:        false,
		}, nil
	}

	log.Printf("🎯 [CHUNKED TRANSLATION] SRT có %d entries (> %d), sử dụng CHUNKED translation", totalEntries, strategy.MaxChunkSize)

	// Chia SRT thành chunks
	chunks, err := t.splitSRTIntoChunks(string(srtContent), strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to split SRT into chunks: %v", err)
	}

	log.Printf("✂️ [CHUNKED TRANSLATION] Đã chia SRT thành %d chunks", len(chunks))

	// Xử lý chunks với concurrent processing
	results, err := t.processChunksConcurrent(chunks, apiKey, modelName, targetLanguage, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to process chunks: %v", err)
	}

	// Ghép chunks lại
	mergedContent, err := t.mergeChunks(results, entries, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to merge chunks: %v", err)
	}

	// Validate kết quả cuối cùng
	if err := t.validateFinalResult(mergedContent, totalEntries); err != nil {
		log.Printf("⚠️ [CHUNKED TRANSLATION] Warning: Final result validation failed: %v", err)
		// Không return error, chỉ log warning
	}

	processingTime := time.Since(startTime)
	log.Printf("🏁 [CHUNKED TRANSLATION] Chunked translation hoàn thành trong %v", processingTime)

	return &ChunkedTranslationResult{
		TranslatedContent: mergedContent,
		ChunksProcessed:   len(results),
		TotalChunks:       len(chunks),
		ProcessingTime:    processingTime,
		HasOverlap:        strategy.OverlapSize > 0,
	}, nil
}

// splitSRTIntoChunks chia SRT thành các chunks nhỏ
func (t *SRTChunkedTranslator) splitSRTIntoChunks(
	srtContent string,
	strategy *SRTChunkingStrategy,
) ([]*SRTChunk, error) {
	entries, err := parseSRT(srtContent)
	if err != nil {
		return nil, err
	}

	var chunks []*SRTChunk
	chunkIndex := 0

	for i := 0; i < len(entries); i += strategy.MaxChunkSize - strategy.OverlapSize {
		endIndex := i + strategy.MaxChunkSize
		if endIndex > len(entries) {
			endIndex = len(entries)
		}

		// Tạo chunk content
		chunkContent := t.createChunkContent(entries[i:endIndex])

		chunk := &SRTChunk{
			ChunkID:    chunkIndex,
			StartIndex: i,
			EndIndex:   endIndex,
			Content:    chunkContent,
			EntryCount: endIndex - i,
			Processed:  false,
		}
		chunks = append(chunks, chunk)
		chunkIndex++
	}

	return chunks, nil
}

// createChunkContent tạo nội dung cho một chunk
func (t *SRTChunkedTranslator) createChunkContent(entries []SRTEntry) string {
	var result strings.Builder
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("%d\n", entry.Index))
		result.WriteString(fmt.Sprintf("%s --> %s\n", formatTime(entry.Start), formatTime(entry.End)))
		result.WriteString(entry.Text + "\n\n")
	}
	return result.String()
}

// processChunksConcurrent xử lý chunks với concurrent processing
func (t *SRTChunkedTranslator) processChunksConcurrent(
	chunks []*SRTChunk,
	apiKey, modelName, targetLanguage string,
	strategy *SRTChunkingStrategy,
) ([]*SRTChunk, error) {
	log.Printf("🚀 [CHUNKED TRANSLATION] Bắt đầu xử lý %d chunks với concurrent processing (max: %d)", len(chunks), strategy.MaxConcurrent)
	log.Printf("🔧 [CHUNKED TRANSLATION] Strategy: chunk_size=%d, overlap=%d, concurrent=%d, timeout=%v",
		strategy.MaxChunkSize, strategy.OverlapSize, strategy.MaxConcurrent, strategy.TimeoutPerChunk)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, strategy.MaxConcurrent)
	results := make([]*SRTChunk, len(chunks))
	var resultMutex sync.Mutex

	// Khởi động workers
	for _, chunk := range chunks {
		wg.Add(1)
		go func(chunk *SRTChunk, index int) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			log.Printf("🔄 [CHUNKED TRANSLATION] Worker bắt đầu xử lý chunk %d (index: %d)", chunk.ChunkID, index)

			// Xử lý chunk với retry logic
			result := t.processSingleChunkWithRetry(chunk, apiKey, modelName, targetLanguage, strategy)

			// Lưu kết quả thread-safe
			resultMutex.Lock()
			results[index] = result
			resultMutex.Unlock()

			if result.Error != nil {
				log.Printf("❌ [CHUNKED TRANSLATION] Chunk %d failed: %v", chunk.ChunkID, result.Error)
			} else {
				log.Printf("✅ [CHUNKED TRANSLATION] Chunk %d completed successfully", chunk.ChunkID)
			}
		}(chunk, chunk.ChunkID)
	}

	log.Printf("⏳ [CHUNKED TRANSLATION] Đang chờ tất cả %d workers hoàn thành...", len(chunks))
	wg.Wait()
	log.Printf("🎯 [CHUNKED TRANSLATION] Tất cả workers đã hoàn thành!")

	// Kiểm tra lỗi
	var failedChunks []int
	for i, result := range results {
		if result.Error != nil {
			failedChunks = append(failedChunks, i)
		}
	}

	if len(failedChunks) > 0 {
		log.Printf("⚠️ [CHUNKED TRANSLATION] %d chunks failed: %v", len(failedChunks), failedChunks)

		// Thử retry với chunk size nhỏ hơn
		log.Printf("🔄 [CHUNKED TRANSLATION] Thử retry failed chunks với chunk size nhỏ hơn...")
		if err := t.retryFailedChunksWithSmallerSize(results, failedChunks, apiKey, modelName, targetLanguage, strategy); err != nil {
			return nil, fmt.Errorf("failed to retry failed chunks: %v", err)
		}
		log.Printf("✅ [CHUNKED TRANSLATION] Retry completed")
	}

	log.Printf("🏁 [CHUNKED TRANSLATION] Concurrent processing hoàn thành cho %d chunks", len(chunks))
	return results, nil
}

// processSingleChunkWithRetry xử lý một chunk với retry logic
func (t *SRTChunkedTranslator) processSingleChunkWithRetry(
	chunk *SRTChunk,
	apiKey, modelName, targetLanguage string,
	strategy *SRTChunkingStrategy,
) *SRTChunk {
	for attempt := 0; attempt <= strategy.RetryAttempts; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying chunk %d, attempt %d/%d", chunk.ChunkID, attempt, strategy.RetryAttempts)
		}

		// Tạo context với timeout
		ctx, cancel := context.WithTimeout(context.Background(), strategy.TimeoutPerChunk)

		// Xử lý chunk
		result, err := t.processSingleChunk(ctx, chunk, apiKey, modelName, targetLanguage)
		cancel()

		if err == nil {
			chunk.Result = result
			chunk.Processed = true
			chunk.Error = nil
			log.Printf("Chunk %d processed successfully", chunk.ChunkID)
			return chunk
		}

		chunk.Error = err
		chunk.RetryCount = attempt

		if attempt < strategy.RetryAttempts {
			// Chờ một chút trước khi retry
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	log.Printf("Chunk %d failed after %d attempts: %v", chunk.ChunkID, strategy.RetryAttempts+1, chunk.Error)
	return chunk
}

// processSingleChunk xử lý một chunk đơn lẻ
func (t *SRTChunkedTranslator) processSingleChunk(
	ctx context.Context,
	chunk *SRTChunk,
	apiKey, modelName, targetLanguage string,
) (string, error) {
	// Tạo prompt cho chunk này
	prompt := t.createChunkPrompt(chunk, targetLanguage)

	// Tự động chọn service dựa trên modelName
	if strings.Contains(strings.ToLower(modelName), "gpt") {
		return t.callGPTAPI(ctx, prompt, apiKey, modelName)
	} else {
		return t.callGeminiAPI(ctx, prompt, apiKey, modelName)
	}
}

// createChunkPrompt tạo prompt cho chunk
func (t *SRTChunkedTranslator) createChunkPrompt(chunk *SRTChunk, targetLanguage string) string {
	languageMap := map[string]string{
		"vi": "Tiếng Việt", "en": "Tiếng Anh", "ja": "Tiếng Nhật",
		"ko": "Tiếng Hàn", "zh": "Tiếng Trung", "fr": "Tiếng Pháp",
		"de": "Tiếng Đức", "es": "Tiếng Tây Ban Nha",
	}

	languageName := languageMap[targetLanguage]
	if languageName == "" {
		languageName = "Tiếng Việt"
	}

	// Sử dụng prompt giống hệt logic cũ để đảm bảo tương thích
	return fmt.Sprintf(`Hãy dịch phần SRT sau sang %s, tối ưu hóa đặc biệt cho Text-to-Speech (TTS).
Mục tiêu cuối cùng là bản dịch khi được đọc lên phải vừa vặn một cách tự nhiên trong khoảng thời gian cho phép, đồng thời phản ánh đúng sắc thái và mối quan hệ của nhân vật qua cách xưng hô.
TUÂN THỦ NGHIÊM NGẶT CÁC QUY TẮC SAU:
QUY TẮC 1: TIMESTAMP VÀ SỐ THỨ TỰ LÀ BẤT BIẾN
Giữ nguyên 100%% số thứ tự và dòng thời gian (timestamps) từ file gốc.
TUYỆT ĐỐI KHÔNG được thay đổi, làm tròn, hay "sửa lỗi" thời gian. Đây là quy tắc quan trọng nhất.
QUY TẮC 2: ƯU TIÊN HÀNG ĐẦU LÀ ĐỘ DÀI CÂU DỊCH
Ngắn gọn là Vua: Câu dịch phải đủ ngắn để đọc xong trong khoảng thời gian của timestamp. Đây là ưu tiên cao hơn việc dịch đầy đủ từng chữ.
Chủ động cô đọng ý: Nắm bắt ý chính và diễn đạt lại một cách súc tích nhất có thể trong văn nói. Mạnh dạn loại bỏ các từ phụ không làm thay đổi ý nghĩa cốt lõi.
Áp dụng quy tắc Ký tự/Giây (CPS): Cố gắng giữ cho câu dịch không vượt quá 17 ký tự cho mỗi giây thời lượng.
Ví dụ: Nếu thời lượng là 2 giây, câu dịch nên dài khoảng 34 ký tự.
QUY TẮC 3: DỊCH TỰ NHIÊN, LINH HOẠT VỀ XƯNG HÔ
Ưu tiên sự tự nhiên và phù hợp với ngữ cảnh giao tiếp.
Về cách xưng hô:
Đối với ngôi thứ nhất, "tôi" là lựa chọn mặc định an toàn và phổ biến nhất trong các tình huống trang trọng hoặc với người lạ.
Tuy nhiên, khi bối cảnh là cuộc trò chuyện thân mật, suồng sã giữa bạn bè, người thân hoặc những người ngang hàng, hãy chủ động sử dụng các đại từ tự nhiên hơn như "tao - mày", "tớ - cậu", v.v., để giữ được sự chân thực của lời thoại.
Hạn chế sử dụng đại từ "ta" trừ khi bối cảnh thật sự đặc trưng (nhân vật là vua chúa, thần linh, hoặc có tính cách rất ngạo mạn).
Mục tiêu là làm cho lời thoại chân thực như người Việt đang nói chuyện, chứ không phải là một bản dịch máy móc.
QUY TẮC 4: QUẢ TRẢ VỀ LUÔN LÀ ĐỊNH DẠNG SRT
Kết quả trả về chỉ là nội dung file srt, không thêm bất kỳ 1 ghi chú hay giải thích gì khác.
QUY TẮC 5: Tên nhân vật, hoặc địa danh. ưu tiên để dạng hán việt, ví dụ: Nhị Cẩu, Cúc Hoa, Đại Lang, Lão Tam .... Bắc Kinh, Hồ Nam, Đại Hưng An Lĩnh
QUY TẮC 6: XỬ LÝ ĐẠI TỪ NHÂN XƯNG CÓ LỰA CHỌN
Khi quy tắc xưng hô cung cấp một lựa chọn (ví dụ: 'thầy/cô', 'tôi/em' ...), bạn BẮT BUỘC PHẢI CHỌN MỘT phương án phù hợp nhất với ngữ cảnh của câu thoại đó. TUYỆT ĐỐI KHÔNG được viết cả hai lựa chọn cách nhau bằng dấu gạch chéo trong câu dịch.

KIỂM TRA CUỐI CÙNG:
Trước khi xuất kết quả, hãy tự kiểm tra lại để chắc chắn:
Không có dòng thời gian nào bị sai lệch.
Độ dài câu dịch hợp lý với thời gian hiển thị.
Cách xưng hô ("tôi", "tao", "tớ", "mày"...) tự nhiên và phù hợp với ngữ cảnh của đoạn hội thoại.
Kết quả chỉ luôn là nội dung của file srt. Không thêm bất kỳ nội dung ghi chú hay giải thích nào khác

Phần SRT cần dịch:
%s`, languageName, chunk.Content)
}

// retryFailedChunksWithSmallerSize retry chunks thất bại với size nhỏ hơn
func (t *SRTChunkedTranslator) retryFailedChunksWithSmallerSize(
	results []*SRTChunk,
	failedChunkIndices []int,
	apiKey, modelName, targetLanguage string,
	strategy *SRTChunkingStrategy,
) error {
	// Giảm chunk size cho lần retry
	smallerStrategy := *strategy
	smallerStrategy.MaxChunkSize = strategy.MaxChunkSize / 2
	if smallerStrategy.MaxChunkSize < 10 {
		smallerStrategy.MaxChunkSize = 10 // Không nhỏ hơn 10
	}

	log.Printf("Retrying failed chunks with smaller size: %d", smallerStrategy.MaxChunkSize)

	for _, index := range failedChunkIndices {
		chunk := results[index]
		if chunk.Error == nil {
			continue
		}

		// Chia chunk thành chunks nhỏ hơn
		smallerChunks, err := t.splitSRTIntoChunks(chunk.Content, &smallerStrategy)
		if err != nil {
			log.Printf("Failed to split failed chunk %d: %v", chunk.ChunkID, err)
			continue
		}

		// Xử lý chunks nhỏ hơn
		smallerResults, err := t.processChunksConcurrent(smallerChunks, apiKey, modelName, targetLanguage, &smallerStrategy)
		if err != nil {
			log.Printf("Failed to process smaller chunks for failed chunk %d: %v", chunk.ChunkID, err)
			continue
		}

		// Ghép chunks nhỏ hơn lại
		mergedContent, err := t.mergeChunks(smallerResults, nil, &smallerStrategy)
		if err != nil {
			log.Printf("Failed to merge smaller chunks for failed chunk %d: %v", chunk.ChunkID, err)
			continue
		}

		// Cập nhật kết quả
		chunk.Result = mergedContent
		chunk.Processed = true
		chunk.Error = nil
		log.Printf("Successfully retried failed chunk %d with smaller size", chunk.ChunkID)
	}

	return nil
}

// mergeChunks ghép các chunks lại thành SRT hoàn chỉnh
func (t *SRTChunkedTranslator) mergeChunks(
	chunks []*SRTChunk,
	originalEntries []SRTEntry,
	strategy *SRTChunkingStrategy,
) (string, error) {
	var mergedEntries []SRTEntry
	var overlapMap map[int]bool

	if strategy.OverlapSize > 0 {
		overlapMap = make(map[int]bool)
	}

	for i, chunk := range chunks {
		if !chunk.Processed {
			return "", fmt.Errorf("chunk %d not processed: %v", chunk.ChunkID, chunk.Error)
		}

		// Parse kết quả chunk
		chunkEntries, err := parseSRT(chunk.Result)
		if err != nil {
			return "", fmt.Errorf("failed to parse chunk %d result: %v", chunk.ChunkID, err)
		}

		// Xử lý overlap với chunk trước (chỉ khi có overlap)
		if i > 0 && strategy.OverlapSize > 0 {
			chunkEntries = t.handleOverlap(chunkEntries, chunks[i-1], overlapMap)
		}

		mergedEntries = append(mergedEntries, chunkEntries...)
	}

	// Tạo SRT content
	return t.createSRTFromEntries(mergedEntries), nil
}

// handleOverlap xử lý overlap giữa các chunks
func (t *SRTChunkedTranslator) handleOverlap(
	currentEntries []SRTEntry,
	previousChunk *SRTChunk,
	overlapMap map[int]bool,
) []SRTEntry {
	if !previousChunk.Processed || len(currentEntries) == 0 {
		return currentEntries
	}

	// Parse previous chunk result
	previousEntries, err := parseSRT(previousChunk.Result)
	if err != nil {
		log.Printf("Warning: failed to parse previous chunk for overlap handling: %v", err)
		return currentEntries
	}

	// Tìm entries overlap
	var filteredEntries []SRTEntry
	for _, entry := range currentEntries {
		// Kiểm tra xem entry này có bị overlap không
		isOverlap := false
		for _, prevEntry := range previousEntries {
			if entry.Index == prevEntry.Index {
				isOverlap = true
				overlapMap[entry.Index] = true
				break
			}
		}

		if !isOverlap {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	return filteredEntries
}

// createSRTFromEntries tạo SRT content từ entries
func (t *SRTChunkedTranslator) createSRTFromEntries(entries []SRTEntry) string {
	var result strings.Builder
	for i, entry := range entries {
		result.WriteString(fmt.Sprintf("%d\n", i+1))
		result.WriteString(fmt.Sprintf("%s --> %s\n", formatTime(entry.Start), formatTime(entry.End)))
		result.WriteString(entry.Text + "\n\n")
	}
	return result.String()
}

// validateFinalResult validate kết quả cuối cùng
func (t *SRTChunkedTranslator) validateFinalResult(mergedContent string, expectedEntryCount int) error {
	entries, err := parseSRT(mergedContent)
	if err != nil {
		return fmt.Errorf("failed to parse merged content: %v", err)
	}

	actualCount := len(entries)
	if abs(actualCount-expectedEntryCount) > 5 {
		return fmt.Errorf("entry count mismatch: expected %d, got %d", expectedEntryCount, actualCount)
	}

	// Kiểm tra timing continuity
	if err := t.validateTimingContinuity(entries); err != nil {
		return fmt.Errorf("timing continuity validation failed: %v", err)
	}

	return nil
}

// validateTimingContinuity kiểm tra tính liên tục của timing
func (t *SRTChunkedTranslator) validateTimingContinuity(entries []SRTEntry) error {
	if len(entries) < 2 {
		return nil
	}

	for i := 1; i < len(entries); i++ {
		prevEnd := entries[i-1].End
		currStart := entries[i].Start

		// Kiểm tra xem timing có hợp lý không
		if prevEnd >= currStart {
			log.Printf("Warning: timing issue at entry %d: prev end %f >= curr start %f",
				i, prevEnd, currStart)
		}
	}

	return nil
}

// abs helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// callGPTAPI gọi GPT API với context timeout
func (t *SRTChunkedTranslator) callGPTAPI(ctx context.Context, prompt, apiKey, modelName string) (string, error) {
	// Gọi GPT API
	reqBody := GPTRequest{
		Model: modelName,
		Messages: []GPTMessage{
			{Role: "user", Content: prompt},
		},
	}

	// Gọi API với context timeout
	url := "https://api.openai.com/v1/chat/completions"
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var gptResp GPTResponse
	json.Unmarshal(respBody, &gptResp)

	if len(gptResp.Choices) > 0 {
		translatedContent := gptResp.Choices[0].Message.Content

		// Clean up response giống hệt logic cũ
		translatedContent = t.cleanupGPTResponse(translatedContent)

		return translatedContent, nil
	}

	return "", fmt.Errorf("no response from API")
}

// callGeminiAPI gọi Gemini API với context timeout
func (t *SRTChunkedTranslator) callGeminiAPI(ctx context.Context, prompt, apiKey, modelName string) (string, error) {
	// Gọi Gemini API
	translatedContent, err := GenerateWithGemini(prompt, apiKey, modelName)
	if err != nil {
		return "", fmt.Errorf("Gemini API call failed: %v", err)
	}

	// Clean up response giống hệt logic cũ
	translatedContent = t.cleanupGeminiResponse(translatedContent)

	return translatedContent, nil
}

// cleanupGPTResponse clean up GPT response giống hệt logic cũ
func (t *SRTChunkedTranslator) cleanupGPTResponse(translatedContent string) string {
	// Clean up the response - remove any extra text that might be added by GPT
	translatedContent = strings.TrimSpace(translatedContent)

	// Remove markdown code blocks if present
	translatedContent = strings.TrimPrefix(translatedContent, "```")
	translatedContent = strings.TrimSuffix(translatedContent, "```")
	translatedContent = strings.TrimSpace(translatedContent)

	// If GPT added any prefix or explanation, try to extract just the SRT content
	if strings.Contains(translatedContent, "1\n") {
		// Find the start of the SRT content
		startIndex := strings.Index(translatedContent, "1\n")
		if startIndex != -1 {
			translatedContent = translatedContent[startIndex:]
		}
	}

	// Remove any explanatory text that might be added after the SRT content
	endMarkers := []string{
		"\n\n**Giải thích",
		"\n\nGiải thích",
		"\n\n**",
		"\n\nTôi đã",
		"\n\nHy vọng",
		"\n\nBản dịch",
		"\n\n---",
		"\n\nNote:",
		"\n\nLưu ý:",
		"\n\n```",
		"```",
	}

	for _, marker := range endMarkers {
		if index := strings.Index(translatedContent, marker); index != -1 {
			translatedContent = strings.TrimSpace(translatedContent[:index])
			break
		}
	}

	// Final cleanup - ensure we only have valid SRT content
	lines := strings.Split(translatedContent, "\n")
	var cleanLines []string
	inSRTContent := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Start SRT content when we see a number
		if t.isNumeric(line) {
			inSRTContent = true
		}

		// Stop if we see explanatory text
		if inSRTContent && (strings.Contains(line, "**") ||
			strings.Contains(line, "Giải thích") ||
			strings.Contains(line, "Tôi đã") ||
			strings.Contains(line, "Hy vọng") ||
			strings.Contains(line, "Bản dịch") ||
			strings.Contains(line, "Note:") ||
			strings.Contains(line, "Lưu ý:") ||
			strings.Contains(line, "```")) {
			break
		}

		if inSRTContent {
			cleanLines = append(cleanLines, line)
		}
	}

	if len(cleanLines) > 0 {
		translatedContent = strings.Join(cleanLines, "\n")
	}

	return translatedContent
}

// cleanupGeminiResponse clean up Gemini response giống hệt logic cũ
func (t *SRTChunkedTranslator) cleanupGeminiResponse(translatedContent string) string {
	// Clean up response giống logic cũ
	translatedContent = strings.TrimSpace(translatedContent)

	// Nếu Gemini thêm prefix, extract chỉ SRT content
	if strings.Contains(translatedContent, "1\n") {
		startIndex := strings.Index(translatedContent, "1\n")
		if startIndex != -1 {
			translatedContent = translatedContent[startIndex:]
		}
	}

	// Remove any explanatory text that might be added after the SRT content
	endMarkers := []string{
		"\n\n**Giải thích",
		"\n\nGiải thích",
		"\n\n**",
		"\n\nTôi đã",
		"\n\nHy vọng",
		"\n\nBản dịch",
		"\n\n---",
		"\n\nNote:",
		"\n\nLưu ý:",
		"\n\n```",
		"```",
	}

	for _, marker := range endMarkers {
		if index := strings.Index(translatedContent, marker); index != -1 {
			translatedContent = strings.TrimSpace(translatedContent[:index])
			break
		}
	}

	// Final cleanup - ensure we only have valid SRT content
	lines := strings.Split(translatedContent, "\n")
	var cleanLines []string
	inSRTContent := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Start SRT content when we see a number
		if t.isNumeric(line) {
			inSRTContent = true
		}

		// Stop if we see explanatory text
		if inSRTContent && (strings.Contains(line, "**") ||
			strings.Contains(line, "Giải thích") ||
			strings.Contains(line, "Tôi đã") ||
			strings.Contains(line, "Hy vọng") ||
			strings.Contains(line, "Bản dịch") ||
			strings.Contains(line, "Note:") ||
			strings.Contains(line, "Lưu ý:") ||
			strings.Contains(line, "```")) {
			break
		}

		if inSRTContent {
			cleanLines = append(cleanLines, line)
		}
	}

	if len(cleanLines) > 0 {
		translatedContent = strings.Join(cleanLines, "\n")
	}

	return translatedContent
}

// isNumeric helper function
func (t *SRTChunkedTranslator) isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// TranslateSRTWithChunkingWrapper wrapper function để tích hợp với logic cũ
// Hỗ trợ cả GPT và Gemini dựa trên service config
func TranslateSRTWithChunkingWrapper(srtFilePath, apiKey, modelName, targetLanguage string) (string, error) {
	// Khởi tạo chunked translator
	translator := GetSRTChunkedTranslator()

	// Sử dụng strategy mặc định
	strategy := &SRTChunkingStrategy{
		MaxChunkSize:    50, // 50 câu mỗi chunk
		OverlapSize:     0,  // 0 câu overlap
		MaxConcurrent:   5,  // 5 chunks đồng thời
		TimeoutPerChunk: 60 * time.Second,
		RetryAttempts:   2,
	}

	// Gọi chunked translation
	result, err := translator.TranslateSRTWithChunking(srtFilePath, apiKey, modelName, targetLanguage, strategy)
	if err != nil {
		return "", err
	}

	return result.TranslatedContent, nil
}

// TranslateSRTWithContextAwareness wrapper function mới với context awareness
// Hỗ trợ cả GPT và Gemini dựa trên service config
func TranslateSRTWithContextAwareness(srtFilePath, apiKey, modelName, targetLanguage string) (string, error) {
	log.Printf("🚀 [CONTEXT AWARE TRANSLATION] Bắt đầu context-aware translation cho %s", srtFilePath)

	// Bước 1: Phân tích ngữ cảnh (một lần gọi API duy nhất)
	contextAnalyzer := NewContextAnalyzer(apiKey, modelName)
	contextResult, err := contextAnalyzer.AnalyzeSRTContext(srtFilePath, targetLanguage)
	if err != nil {
		log.Printf("⚠️ [CONTEXT AWARE TRANSLATION] Context analysis failed, fallback to chunked translation: %v", err)
		// Fallback to chunked translation nếu context analysis thất bại
		return TranslateSRTWithChunkingWrapper(srtFilePath, apiKey, modelName, targetLanguage)
	}

	// Bước 2: Tạo prompt mẫu với context awareness
	contextAwarePrompt := contextAnalyzer.GenerateContextAwarePrompt(contextResult, targetLanguage)

	// Bước 3: Chia file SRT thành chunks
	chunks, err := SplitSRTIntoChunksForContextAware(srtFilePath, 50, 0) // 50 câu mỗi chunk, 0 overlap
	if err != nil {
		return "", fmt.Errorf("failed to split SRT into chunks: %v", err)
	}

	log.Printf("📊 [CONTEXT AWARE TRANSLATION] Đã chia SRT thành %d chunks", len(chunks))

	// Bước 4: Xử lý chunks với context-aware prompts
	results, err := processChunksWithContextAwareness(chunks, contextAwarePrompt, apiKey, modelName, 5) // 5 concurrent
	if err != nil {
		return "", fmt.Errorf("failed to process chunks with context awareness: %v", err)
	}

	// Bước 5: Ghép chunks lại
	mergedContent, err := mergeChunksWithContextAwareness(results, chunks)
	if err != nil {
		return "", fmt.Errorf("failed to merge chunks: %v", err)
	}

	log.Printf("✅ [CONTEXT AWARE TRANSLATION] Context-aware translation hoàn thành!")
	return mergedContent, nil
}

// SplitSRTIntoChunksForContextAware chia SRT thành chunks cho context-aware translation
func SplitSRTIntoChunksForContextAware(srtFilePath string, maxChunkSize, overlapSize int) ([]*SRTChunk, error) {
	// Tạo temporary translator để sử dụng createChunkContent method
	tempTranslator := &SRTChunkedTranslator{}
	srtContent, err := os.ReadFile(srtFilePath)
	if err != nil {
		return nil, err
	}

	entries, err := parseSRT(string(srtContent))
	if err != nil {
		return nil, err
	}

	var chunks []*SRTChunk
	chunkIndex := 0

	for i := 0; i < len(entries); i += maxChunkSize - overlapSize {
		endIndex := i + maxChunkSize
		if endIndex > len(entries) {
			endIndex = len(entries)
		}

		chunkContent := tempTranslator.createChunkContent(entries[i:endIndex])

		chunk := &SRTChunk{
			ChunkID:    chunkIndex,
			StartIndex: i,
			EndIndex:   endIndex,
			Content:    chunkContent,
			EntryCount: endIndex - i,
			Processed:  false,
		}
		chunks = append(chunks, chunk)
		chunkIndex++
	}

	return chunks, nil
}

// processChunksWithContextAwareness xử lý chunks với context-aware prompts
func processChunksWithContextAwareness(chunks []*SRTChunk, contextAwarePrompt, apiKey, modelName string, maxConcurrent int) ([]*SRTChunk, error) {
	log.Printf("🚀 [CONTEXT AWARE TRANSLATION] Bắt đầu xử lý %d chunks với context awareness (max concurrent: %d)", len(chunks), maxConcurrent)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrent)
	results := make([]*SRTChunk, len(chunks))
	var resultMutex sync.Mutex

	// Khởi động workers
	for _, chunk := range chunks {
		wg.Add(1)
		go func(chunk *SRTChunk, index int) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			log.Printf("🔄 [CONTEXT AWARE TRANSLATION] Worker bắt đầu xử lý chunk %d (index: %d)", chunk.ChunkID, index)

			// Tạo prompt cho chunk này với context awareness
			chunkPrompt := strings.Replace(contextAwarePrompt, "{{SRT_CONTENT}}", chunk.Content, 1)

			// Xử lý chunk với context-aware prompt
			result := processSingleChunkWithContextAwareness(chunk, chunkPrompt, apiKey, modelName)

			// Lưu kết quả thread-safe
			resultMutex.Lock()
			results[index] = result
			resultMutex.Unlock()

			if result.Error != nil {
				log.Printf("❌ [CONTEXT AWARE TRANSLATION] Chunk %d failed: %v", chunk.ChunkID, result.Error)
			} else {
				log.Printf("✅ [CONTEXT AWARE TRANSLATION] Chunk %d completed successfully", chunk.ChunkID)
			}
		}(chunk, chunk.ChunkID)
	}

	log.Printf("⏳ [CONTEXT AWARE TRANSLATION] Đang chờ tất cả %d workers hoàn thành...", len(chunks))
	wg.Wait()
	log.Printf("🎯 [CONTEXT AWARE TRANSLATION] Tất cả workers đã hoàn thành!")

	return results, nil
}

// processSingleChunkWithContextAwareness xử lý một chunk với context-aware prompt
func processSingleChunkWithContextAwareness(chunk *SRTChunk, chunkPrompt, apiKey, modelName string) *SRTChunk {
	// Tự động chọn service dựa trên modelName
	var translatedContent string
	var err error

	if strings.Contains(strings.ToLower(modelName), "gpt") {
		translatedContent, err = callGPTAPIForChunk(chunkPrompt, apiKey, modelName)
	} else {
		translatedContent, err = callGeminiAPIForChunk(chunkPrompt, apiKey, modelName)
	}

	if err != nil {
		chunk.Error = err
		chunk.Processed = false
	} else {
		chunk.Result = translatedContent
		chunk.Processed = true
		chunk.Error = nil
	}

	return chunk
}

// callGPTAPIForChunk gọi GPT API cho một chunk
func callGPTAPIForChunk(prompt, apiKey, modelName string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	requestBody := map[string]interface{}{
		"model":       modelName,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.1,
		"max_tokens":  4000,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GPT API request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GPT API error: %s - %s", resp.Status, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse GPT response: %v", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in GPT response")
	}

	translatedContent := response.Choices[0].Message.Content

	// Remove markdown code blocks if present - simple approach
	translatedContent = strings.TrimPrefix(translatedContent, "```srt")
	translatedContent = strings.TrimPrefix(translatedContent, "```")
	translatedContent = strings.TrimSuffix(translatedContent, "```")
	// translatedContent = strings.TrimSpace(translatedContent)

	// Remove any remaining "srt" at the beginning
	if strings.HasPrefix(translatedContent, "srt") {
		translatedContent = strings.TrimPrefix(translatedContent, "srt")
		//translatedContent = strings.TrimSpace(translatedContent)
	}

	return translatedContent, nil
}

// callGeminiAPIForChunk gọi Gemini API cho một chunk
func callGeminiAPIForChunk(prompt, apiKey, modelName string) (string, error) {
	// Sử dụng service Gemini có sẵn
	return GenerateWithGemini(prompt, apiKey, modelName)
}

// mergeChunksWithContextAwareness ghép chunks lại với context awareness
func mergeChunksWithContextAwareness(results []*SRTChunk, originalChunks []*SRTChunk) (string, error) {
	var mergedBuilder strings.Builder
	var seenEntries map[int]bool = make(map[int]bool)
	var entryCounter int = 1

	for _, result := range results {
		if result.Error != nil {
			return "", fmt.Errorf("chunk %d failed: %v", result.ChunkID, result.Error)
		}

		if !result.Processed {
			return "", fmt.Errorf("chunk %d not processed", result.ChunkID)
		}

		// Parse chunk result để xử lý overlap
		chunkEntries, err := parseSRT(result.Result)
		if err != nil {
			return "", fmt.Errorf("failed to parse chunk %d result: %v", result.ChunkID, err)
		}

		// Xử lý từng entry trong chunk
		for _, entry := range chunkEntries {
			// Kiểm tra xem entry này đã được xử lý chưa (dựa trên index)
			if !seenEntries[entry.Index] {
				seenEntries[entry.Index] = true

				// Ghi entry với số thứ tự mới
				mergedBuilder.WriteString(fmt.Sprintf("%d\n", entryCounter))
				mergedBuilder.WriteString(fmt.Sprintf("%s --> %s\n", formatTime(entry.Start), formatTime(entry.End)))
				mergedBuilder.WriteString(entry.Text + "\n\n")

				entryCounter++
			}
		}
	}

	return mergedBuilder.String(), nil
}
