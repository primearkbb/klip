package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/john/klip/internal/api"
	"github.com/john/klip/internal/storage"
)

// StreamingState represents the state of a streaming operation
type StreamingState struct {
	Active        bool
	Buffer        strings.Builder
	ChunkCount    int
	StartTime     time.Time
	LastChunkTime time.Time
	TotalTokens   int
	BytesReceived int64
	Error         error
	Interrupted   bool
}

// NewStreamingState creates a new streaming state
func NewStreamingState() *StreamingState {
	return &StreamingState{
		StartTime: time.Now(),
	}
}

// Start marks the streaming as started
func (ss *StreamingState) Start() {
	ss.Active = true
	ss.StartTime = time.Now()
	ss.LastChunkTime = ss.StartTime
	ss.Buffer.Reset()
	ss.ChunkCount = 0
	ss.TotalTokens = 0
	ss.BytesReceived = 0
	ss.Error = nil
	ss.Interrupted = false
}

// AddChunk adds a chunk to the streaming buffer
func (ss *StreamingState) AddChunk(content string) {
	if !ss.Active {
		return
	}
	
	ss.Buffer.WriteString(content)
	ss.ChunkCount++
	ss.LastChunkTime = time.Now()
	ss.BytesReceived += int64(len(content))
}

// Complete marks the streaming as completed
func (ss *StreamingState) Complete() string {
	ss.Active = false
	return ss.Buffer.String()
}

// Interrupt marks the streaming as interrupted
func (ss *StreamingState) Interrupt() {
	ss.Active = false
	ss.Interrupted = true
}

// SetError sets an error and stops streaming
func (ss *StreamingState) SetError(err error) {
	ss.Active = false
	ss.Error = err
}

// GetContent returns the current content
func (ss *StreamingState) GetContent() string {
	return ss.Buffer.String()
}

// GetDuration returns the duration of the streaming
func (ss *StreamingState) GetDuration() time.Duration {
	if ss.StartTime.IsZero() {
		return 0
	}
	if ss.Active {
		return time.Since(ss.StartTime)
	}
	return ss.LastChunkTime.Sub(ss.StartTime)
}

// GetSpeed returns the streaming speed in characters per second
func (ss *StreamingState) GetSpeed() float64 {
	duration := ss.GetDuration()
	if duration == 0 {
		return 0
	}
	return float64(ss.Buffer.Len()) / duration.Seconds()
}

// StreamingManager handles streaming operations
type StreamingManager struct {
	model         *Model
	currentStream *StreamingState
	cancelFunc    context.CancelFunc
	progressChan  chan StreamProgressMsg
}

// NewStreamingManager creates a new streaming manager
func NewStreamingManager(model *Model) *StreamingManager {
	return &StreamingManager{
		model:        model,
		progressChan: make(chan StreamProgressMsg, 100),
	}
}

// StreamProgressMsg represents streaming progress
type StreamProgressMsg struct {
	Content       string
	ChunkIndex    int
	TotalChunks   int
	BytesReceived int64
	Duration      time.Duration
	Speed         float64
	Done          bool
	Error         error
}

// StartStreaming starts a streaming request
func (sm *StreamingManager) StartStreaming(request *api.ChatRequest) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if sm.model.apiClient == nil {
			return apiErrorMsg{fmt.Errorf("no API client available")}
		}

		// Create new streaming state
		sm.currentStream = NewStreamingState()
		sm.currentStream.Start()

		// Create cancellable context
		ctx, cancel := context.WithCancel(sm.model.ctx)
		sm.cancelFunc = cancel

		// Set up interrupt handling
		go func() {
			select {
			case <-sm.model.chatState.InterruptChannel:
				sm.InterruptStreaming()
			case <-ctx.Done():
			}
		}()

		// Start the streaming request
		chunkChan, errChan := sm.model.apiClient.ChatStream(ctx, request)

		// Handle streaming in a goroutine
		go sm.handleStreamingResponse(ctx, chunkChan, errChan)

		// Update UI to show streaming started
		sm.model.chatState.IsStreaming = true
		sm.model.chatState.WaitingForAPI = true

		return apiStreamStartMsg{}
	})
}

// handleStreamingResponse handles the streaming response
func (sm *StreamingManager) handleStreamingResponse(ctx context.Context, chunkChan <-chan api.StreamChunk, errChan <-chan error) {
	defer func() {
		sm.model.chatState.IsStreaming = false
		sm.model.chatState.WaitingForAPI = false
	}()

	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				// Channel closed, streaming finished
				sm.finishStreaming()
				return
			}

			// Add chunk to buffer
			sm.currentStream.AddChunk(chunk.Content)

			// Send progress update
			progress := StreamProgressMsg{
				Content:       chunk.Content,
				ChunkIndex:    sm.currentStream.ChunkCount,
				BytesReceived: sm.currentStream.BytesReceived,
				Duration:      sm.currentStream.GetDuration(),
				Speed:         sm.currentStream.GetSpeed(),
				Done:          chunk.Done,
			}

			// Send update to UI
			tea.Batch(func() tea.Msg {
				return apiStreamChunkMsg{chunk.Content}
			})()

			// Send progress update
			select {
			case sm.progressChan <- progress:
			default:
				// Channel full, skip this progress update
			}

			// Check if streaming is complete
			if chunk.Done {
				sm.finishStreaming()
				return
			}

		case err, ok := <-errChan:
			if !ok {
				return
			}

			// Handle error
			sm.currentStream.SetError(err)
			tea.Batch(func() tea.Msg {
				return apiErrorMsg{err}
			})()
			return

		case <-ctx.Done():
			// Context cancelled (likely interrupted)
			sm.currentStream.Interrupt()
			return
		}
	}
}

// finishStreaming completes the streaming operation
func (sm *StreamingManager) finishStreaming() {
	if sm.currentStream == nil {
		return
	}

	content := sm.currentStream.Complete()

	// Create assistant message
	assistantMsg := api.Message{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
	}

	// Add to chat history
	sm.model.chatState.AddMessage(assistantMsg)

	// Log the message (convert to storage format)
	if sm.model.storage != nil && sm.model.storage.ChatLogger != nil {
		go func() {
			storageMsg := storage.Message{
				Role:      assistantMsg.Role,
				Content:   assistantMsg.Content,
				Timestamp: assistantMsg.Timestamp,
			}
			if err := sm.model.storage.ChatLogger.LogMessage(storageMsg); err != nil {
				sm.model.logger.Error("Failed to log assistant message", "error", err)
			}
		}()
	}

	// Log analytics
	if sm.model.storage != nil && sm.model.storage.AnalyticsLogger != nil {
		go func() {
			metrics := api.ResponseMetrics{
				LatencyMs:      sm.currentStream.GetDuration().Milliseconds(),
				TokensInput:    0, // TODO: Calculate from request
				TokensOutput:   len(strings.Fields(content)), // Rough estimate
				ResponseLength: len(content),
			}

			// TODO: Implement API response logging when method is available
			_ = metrics // Avoid unused variable warning
		}()
	}

	// Send completion message
	tea.Batch(func() tea.Msg {
		return apiStreamDoneMsg{}
	})()

	// Clear streaming buffer
	sm.model.chatState.StreamBuffer = ""
}

// InterruptStreaming interrupts the current streaming operation
func (sm *StreamingManager) InterruptStreaming() {
	if sm.cancelFunc != nil {
		sm.cancelFunc()
		sm.cancelFunc = nil
	}

	if sm.currentStream != nil {
		sm.currentStream.Interrupt()
	}

	sm.model.chatState.IsStreaming = false
	sm.model.chatState.WaitingForAPI = false

	// Send interrupt message
	tea.Batch(func() tea.Msg {
		return apiStreamInterruptMsg{}
	})()
}

// IsStreaming returns true if currently streaming
func (sm *StreamingManager) IsStreaming() bool {
	return sm.currentStream != nil && sm.currentStream.Active
}

// GetProgress returns the current streaming progress
func (sm *StreamingManager) GetProgress() *StreamingState {
	return sm.currentStream
}

// Additional streaming message types
type (
	apiStreamStartMsg     struct{}
	apiStreamInterruptMsg struct{}
)

// ProgressTracker tracks progress of long-running operations
type ProgressTracker struct {
	operation    string
	startTime    time.Time
	progress     float64
	status       string
	subStatus    string
	error        error
	completed    bool
	cancelled    bool
	estimatedEnd time.Time
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(operation string) *ProgressTracker {
	return &ProgressTracker{
		operation: operation,
		startTime: time.Now(),
	}
}

// SetProgress updates the progress (0.0 to 1.0)
func (pt *ProgressTracker) SetProgress(progress float64, status string) {
	pt.progress = progress
	pt.status = status

	// Estimate completion time
	if progress > 0 {
		elapsed := time.Since(pt.startTime)
		totalEstimated := time.Duration(float64(elapsed) / progress)
		pt.estimatedEnd = pt.startTime.Add(totalEstimated)
	}
}

// SetSubStatus sets a sub-status message
func (pt *ProgressTracker) SetSubStatus(subStatus string) {
	pt.subStatus = subStatus
}

// SetError sets an error and marks as failed
func (pt *ProgressTracker) SetError(err error) {
	pt.error = err
	pt.completed = true
}

// Complete marks the operation as completed
func (pt *ProgressTracker) Complete() {
	pt.progress = 1.0
	pt.completed = true
}

// Cancel marks the operation as cancelled
func (pt *ProgressTracker) Cancel() {
	pt.cancelled = true
}

// GetElapsed returns the elapsed time
func (pt *ProgressTracker) GetElapsed() time.Duration {
	return time.Since(pt.startTime)
}

// GetETA returns the estimated time to completion
func (pt *ProgressTracker) GetETA() time.Duration {
	if pt.estimatedEnd.IsZero() || pt.completed || pt.cancelled {
		return 0
	}
	
	eta := time.Until(pt.estimatedEnd)
	if eta < 0 {
		return 0
	}
	return eta
}

// IsActive returns true if the operation is still active
func (pt *ProgressTracker) IsActive() bool {
	return !pt.completed && !pt.cancelled
}

// GetProgressPercent returns progress as a percentage
func (pt *ProgressTracker) GetProgressPercent() int {
	return int(pt.progress * 100)
}

// GetStatusText returns a formatted status text
func (pt *ProgressTracker) GetStatusText() string {
	if pt.error != nil {
		return fmt.Sprintf("Error: %v", pt.error)
	}
	
	if pt.cancelled {
		return "Cancelled"
	}
	
	if pt.completed {
		return "Completed"
	}
	
	status := pt.status
	if pt.subStatus != "" {
		status = fmt.Sprintf("%s - %s", status, pt.subStatus)
	}
	
	return status
}

// GetDetailedStatus returns detailed status information
func (pt *ProgressTracker) GetDetailedStatus() string {
	elapsed := pt.GetElapsed()
	
	var parts []string
	parts = append(parts, fmt.Sprintf("Operation: %s", pt.operation))
	parts = append(parts, fmt.Sprintf("Progress: %d%%", pt.GetProgressPercent()))
	parts = append(parts, fmt.Sprintf("Status: %s", pt.GetStatusText()))
	parts = append(parts, fmt.Sprintf("Elapsed: %s", formatDuration(elapsed)))
	
	if eta := pt.GetETA(); eta > 0 {
		parts = append(parts, fmt.Sprintf("ETA: %s", formatDuration(eta)))
	}
	
	return strings.Join(parts, "\n")
}

// TokenCounter provides token counting utilities for streaming
type TokenCounter struct {
	inputTokens  int
	outputTokens int
	totalTokens  int
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{}
}

// AddInputTokens adds input tokens to the count
func (tc *TokenCounter) AddInputTokens(count int) {
	tc.inputTokens += count
	tc.totalTokens += count
}

// AddOutputTokens adds output tokens to the count
func (tc *TokenCounter) AddOutputTokens(count int) {
	tc.outputTokens += count
	tc.totalTokens += count
}

// EstimateTokens estimates token count from text (rough approximation)
func (tc *TokenCounter) EstimateTokens(text string) int {
	// Very rough estimation: ~4 characters per token
	return len(text) / 4
}

// GetInputTokens returns the input token count
func (tc *TokenCounter) GetInputTokens() int {
	return tc.inputTokens
}

// GetOutputTokens returns the output token count
func (tc *TokenCounter) GetOutputTokens() int {
	return tc.outputTokens
}

// GetTotalTokens returns the total token count
func (tc *TokenCounter) GetTotalTokens() int {
	return tc.totalTokens
}

// Reset resets all token counts
func (tc *TokenCounter) Reset() {
	tc.inputTokens = 0
	tc.outputTokens = 0
	tc.totalTokens = 0
}

// StreamingMetrics provides metrics for streaming operations
type StreamingMetrics struct {
	StartTime        time.Time
	EndTime          time.Time
	ChunksReceived   int
	BytesReceived    int64
	TokensReceived   int
	AverageChunkSize float64
	PeakSpeed        float64
	AverageSpeed     float64
	Interrupted      bool
	Error            error
}

// NewStreamingMetrics creates new streaming metrics
func NewStreamingMetrics() *StreamingMetrics {
	return &StreamingMetrics{
		StartTime: time.Now(),
	}
}

// AddChunk records a received chunk
func (sm *StreamingMetrics) AddChunk(size int, tokens int) {
	sm.ChunksReceived++
	sm.BytesReceived += int64(size)
	sm.TokensReceived += tokens
	
	// Update average chunk size
	sm.AverageChunkSize = float64(sm.BytesReceived) / float64(sm.ChunksReceived)
	
	// Calculate current speed
	duration := time.Since(sm.StartTime).Seconds()
	if duration > 0 {
		currentSpeed := float64(sm.BytesReceived) / duration
		if currentSpeed > sm.PeakSpeed {
			sm.PeakSpeed = currentSpeed
		}
		sm.AverageSpeed = currentSpeed
	}
}

// Finish marks the streaming as finished
func (sm *StreamingMetrics) Finish() {
	sm.EndTime = time.Now()
}

// SetError sets an error
func (sm *StreamingMetrics) SetError(err error) {
	sm.Error = err
	sm.Finish()
}

// SetInterrupted marks as interrupted
func (sm *StreamingMetrics) SetInterrupted() {
	sm.Interrupted = true
	sm.Finish()
}

// GetDuration returns the total duration
func (sm *StreamingMetrics) GetDuration() time.Duration {
	if sm.EndTime.IsZero() {
		return time.Since(sm.StartTime)
	}
	return sm.EndTime.Sub(sm.StartTime)
}

// GetThroughput returns throughput in bytes per second
func (sm *StreamingMetrics) GetThroughput() float64 {
	duration := sm.GetDuration().Seconds()
	if duration == 0 {
		return 0
	}
	return float64(sm.BytesReceived) / duration
}