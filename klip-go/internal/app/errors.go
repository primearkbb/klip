package app

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ErrorType categorizes different types of errors
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeNetwork
	ErrorTypeAPI
	ErrorTypeAuth
	ErrorTypeRateLimit
	ErrorTypeStorage
	ErrorTypeConfig
	ErrorTypeValidation
	ErrorTypeTimeout
	ErrorTypeInterrupted
	ErrorTypeDependency
)

// String returns the string representation of ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeNetwork:
		return "Network"
	case ErrorTypeAPI:
		return "API"
	case ErrorTypeAuth:
		return "Authentication"
	case ErrorTypeRateLimit:
		return "Rate Limit"
	case ErrorTypeStorage:
		return "Storage"
	case ErrorTypeConfig:
		return "Configuration"
	case ErrorTypeValidation:
		return "Validation"
	case ErrorTypeTimeout:
		return "Timeout"
	case ErrorTypeInterrupted:
		return "Interrupted"
	case ErrorTypeDependency:
		return "Dependency"
	default:
		return "Unknown"
	}
}

// AppError represents an application-specific error with context
type AppError struct {
	Type        ErrorType
	Message     string
	Cause       error
	Context     map[string]interface{}
	Recoverable bool
	UserMessage string
	Code        string
	Timestamp   time.Time
	RetryAfter  time.Duration
}

// NewAppError creates a new application error
func NewAppError(errorType ErrorType, message string, cause error) *AppError {
	return &AppError{
		Type:      errorType,
		Message:   message,
		Cause:     cause,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// Error implements the error interface
func (ae *AppError) Error() string {
	if ae.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", ae.Type.String(), ae.Message, ae.Cause)
	}
	return fmt.Sprintf("%s: %s", ae.Type.String(), ae.Message)
}

// Unwrap returns the underlying cause error
func (ae *AppError) Unwrap() error {
	return ae.Cause
}

// WithContext adds context information to the error
func (ae *AppError) WithContext(key string, value interface{}) *AppError {
	ae.Context[key] = value
	return ae
}

// WithUserMessage sets a user-friendly message
func (ae *AppError) WithUserMessage(message string) *AppError {
	ae.UserMessage = message
	return ae
}

// WithCode sets an error code
func (ae *AppError) WithCode(code string) *AppError {
	ae.Code = code
	return ae
}

// WithRetryAfter sets a retry delay
func (ae *AppError) WithRetryAfter(duration time.Duration) *AppError {
	ae.RetryAfter = duration
	return ae
}

// MakeRecoverable marks the error as recoverable
func (ae *AppError) MakeRecoverable() *AppError {
	ae.Recoverable = true
	return ae
}

// GetUserMessage returns a user-friendly error message
func (ae *AppError) GetUserMessage() string {
	if ae.UserMessage != "" {
		return ae.UserMessage
	}
	return ae.Message
}

// ErrorHandler handles errors and provides recovery strategies
type ErrorHandler struct {
	model         *Model
	retryStrategy *RetryStrategy
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(model *Model) *ErrorHandler {
	return &ErrorHandler{
		model:         model,
		retryStrategy: NewRetryStrategy(),
	}
}

// HandleError processes an error and determines the appropriate response
func (eh *ErrorHandler) HandleError(err error) tea.Cmd {
	if err == nil {
		return nil
	}

	// Classify the error
	appErr := eh.classifyError(err)

	// Log the error
	eh.logError(appErr)

	// Set error state
	eh.model.setError(appErr, appErr.GetUserMessage(), appErr.Recoverable)

	// Return appropriate command based on error type
	return eh.getErrorResponse(appErr)
}

// classifyError classifies an error into an AppError
func (eh *ErrorHandler) classifyError(err error) *AppError {
	// Check if it's already an AppError
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// Network errors
	if isNetworkError(err) {
		return NewAppError(ErrorTypeNetwork, "Network connectivity issue", err).
			WithUserMessage("Unable to connect to the service. Please check your internet connection.").
			MakeRecoverable()
	}

	// HTTP errors - check for wrapped HTTP response errors
	// This would need to be implemented based on actual HTTP client error types

	// Context errors
	if errors.Is(err, context.Canceled) {
		return NewAppError(ErrorTypeInterrupted, "Operation was cancelled", err).
			WithUserMessage("The operation was interrupted.")
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return NewAppError(ErrorTypeTimeout, "Operation timed out", err).
			WithUserMessage("The operation took too long to complete. Please try again.").
			MakeRecoverable()
	}

	// API-specific errors
	if strings.Contains(err.Error(), "API key") || strings.Contains(err.Error(), "authentication") {
		return NewAppError(ErrorTypeAuth, "Authentication failed", err).
			WithUserMessage("Invalid or missing API key. Please check your credentials.").
			MakeRecoverable()
	}

	if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "quota") {
		return NewAppError(ErrorTypeRateLimit, "Rate limit exceeded", err).
			WithUserMessage("Too many requests. Please wait a moment before trying again.").
			WithRetryAfter(time.Minute).
			MakeRecoverable()
	}

	// Storage errors
	if strings.Contains(err.Error(), "storage") || strings.Contains(err.Error(), "database") {
		return NewAppError(ErrorTypeStorage, "Storage operation failed", err).
			WithUserMessage("Unable to save or retrieve data. Please try again.").
			MakeRecoverable()
	}

	// Default to unknown error
	return NewAppError(ErrorTypeUnknown, "An unexpected error occurred", err).
		WithUserMessage("Something went wrong. Please try again.").
		MakeRecoverable()
}

// classifyHTTPError classifies HTTP response errors
func (eh *ErrorHandler) classifyHTTPError(resp *http.Response) *AppError {
	switch resp.StatusCode {
	case 401:
		return NewAppError(ErrorTypeAuth, "Unauthorized", fmt.Errorf("HTTP %d", resp.StatusCode)).
			WithUserMessage("Authentication failed. Please check your API key.").
			MakeRecoverable()
	case 403:
		return NewAppError(ErrorTypeAuth, "Forbidden", fmt.Errorf("HTTP %d", resp.StatusCode)).
			WithUserMessage("Access denied. Please check your permissions.")
	case 429:
		retryAfter := eh.parseRetryAfter(resp)
		return NewAppError(ErrorTypeRateLimit, "Rate limit exceeded", fmt.Errorf("HTTP %d", resp.StatusCode)).
			WithUserMessage("Too many requests. Please wait before trying again.").
			WithRetryAfter(retryAfter).
			MakeRecoverable()
	case 500:
		return NewAppError(ErrorTypeAPI, "Server error", fmt.Errorf("HTTP %d", resp.StatusCode)).
			WithUserMessage("The server encountered an error. Please try again later.").
			MakeRecoverable()
	case 502, 503, 504:
		return NewAppError(ErrorTypeAPI, "Service unavailable", fmt.Errorf("HTTP %d", resp.StatusCode)).
			WithUserMessage("The service is temporarily unavailable. Please try again later.").
			MakeRecoverable()
	default:
		return NewAppError(ErrorTypeAPI, "API error", fmt.Errorf("HTTP %d", resp.StatusCode)).
			WithUserMessage(fmt.Sprintf("API request failed with status %d.", resp.StatusCode)).
			MakeRecoverable()
	}
}

// parseRetryAfter parses the Retry-After header
func (eh *ErrorHandler) parseRetryAfter(resp *http.Response) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return time.Minute // Default retry after 1 minute
	}

	// Try to parse as seconds
	if duration, err := time.ParseDuration(retryAfter + "s"); err == nil {
		return duration
	}

	// Default fallback
	return time.Minute
}

// isNetworkError checks if an error is network-related
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network operation errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for connection errors
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Check for syscall errors
	var syscallErr syscall.Errno
	if errors.As(err, &syscallErr) {
		switch syscallErr {
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ECONNABORTED:
			return true
		}
	}

	// Check error message for common network issues
	errMsg := strings.ToLower(err.Error())
	networkKeywords := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"network unreachable",
		"host unreachable",
		"no route to host",
		"temporary failure",
		"name resolution failed",
	}

	for _, keyword := range networkKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// logError logs an error with appropriate details
func (eh *ErrorHandler) logError(appErr *AppError) {
	if eh.model.logger == nil {
		return
	}

	fields := []interface{}{
		"type", appErr.Type.String(),
		"message", appErr.Message,
		"recoverable", appErr.Recoverable,
		"timestamp", appErr.Timestamp,
	}

	if appErr.Code != "" {
		fields = append(fields, "code", appErr.Code)
	}

	if appErr.RetryAfter > 0 {
		fields = append(fields, "retry_after", appErr.RetryAfter)
	}

	for key, value := range appErr.Context {
		fields = append(fields, key, value)
	}

	if appErr.Cause != nil {
		fields = append(fields, "cause", appErr.Cause)
	}

	eh.model.logger.Error("Application error occurred", fields...)

	// Log to analytics if available
	// TODO: Implement error logging to analytics when the method is available
}

// getErrorResponse returns appropriate command based on error type
func (eh *ErrorHandler) getErrorResponse(appErr *AppError) tea.Cmd {
	switch appErr.Type {
	case ErrorTypeRateLimit:
		if appErr.RetryAfter > 0 {
			return tea.Tick(appErr.RetryAfter, func(t time.Time) tea.Msg {
				return statusMsg{"Ready to retry", 2 * time.Second}
			})
		}
	case ErrorTypeNetwork:
		// Could implement network connectivity check here
		return func() tea.Msg {
			return statusMsg{"Check your internet connection", 5 * time.Second}
		}
	case ErrorTypeAuth:
		// Could prompt for new API key
		return func() tea.Msg {
			return statusMsg{"Please check your API credentials", 5 * time.Second}
		}
	}

	return nil
}

// RetryStrategy handles retry logic with exponential backoff
type RetryStrategy struct {
	maxRetries    int
	baseDelay     time.Duration
	maxDelay      time.Duration
	backoffFactor float64
	jitter        bool
}

// NewRetryStrategy creates a new retry strategy with sensible defaults
func NewRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		maxRetries:    3,
		baseDelay:     time.Second,
		maxDelay:      30 * time.Second,
		backoffFactor: 2.0,
		jitter:        true,
	}
}

// WithMaxRetries sets the maximum number of retries
func (rs *RetryStrategy) WithMaxRetries(max int) *RetryStrategy {
	rs.maxRetries = max
	return rs
}

// WithBaseDelay sets the base delay
func (rs *RetryStrategy) WithBaseDelay(delay time.Duration) *RetryStrategy {
	rs.baseDelay = delay
	return rs
}

// WithMaxDelay sets the maximum delay
func (rs *RetryStrategy) WithMaxDelay(delay time.Duration) *RetryStrategy {
	rs.maxDelay = delay
	return rs
}

// CalculateDelay calculates the delay for a given retry attempt
func (rs *RetryStrategy) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Exponential backoff
	delay := float64(rs.baseDelay) * math.Pow(rs.backoffFactor, float64(attempt-1))
	
	// Apply maximum delay limit
	if delay > float64(rs.maxDelay) {
		delay = float64(rs.maxDelay)
	}

	// Add jitter to avoid thundering herd
	if rs.jitter {
		jitterAmount := delay * 0.1 // 10% jitter
		jitter := (rand.Float64() - 0.5) * 2 * jitterAmount
		delay += jitter
	}

	// Ensure minimum delay
	if delay < float64(rs.baseDelay) {
		delay = float64(rs.baseDelay)
	}

	return time.Duration(delay)
}

// ShouldRetry determines if an error should be retried
func (rs *RetryStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= rs.maxRetries {
		return false
	}

	// Check if it's a retryable error
	appErr, ok := err.(*AppError)
	if !ok {
		return false
	}

	if !appErr.Recoverable {
		return false
	}

	// Don't retry authentication errors
	if appErr.Type == ErrorTypeAuth {
		return false
	}

	// Don't retry validation errors
	if appErr.Type == ErrorTypeValidation {
		return false
	}

	// Don't retry interrupted operations
	if appErr.Type == ErrorTypeInterrupted {
		return false
	}

	return true
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation struct {
	model     *Model
	strategy  *RetryStrategy
	operation func() error
	onRetry   func(attempt int, err error)
	maxRetries int
}

// NewRetryableOperation creates a new retryable operation
func NewRetryableOperation(model *Model, operation func() error) *RetryableOperation {
	return &RetryableOperation{
		model:     model,
		strategy:  NewRetryStrategy(),
		operation: operation,
	}
}

// WithStrategy sets the retry strategy
func (ro *RetryableOperation) WithStrategy(strategy *RetryStrategy) *RetryableOperation {
	ro.strategy = strategy
	return ro
}

// WithOnRetry sets a callback for retry attempts
func (ro *RetryableOperation) WithOnRetry(callback func(int, error)) *RetryableOperation {
	ro.onRetry = callback
	return ro
}

// Execute executes the operation with retry logic
func (ro *RetryableOperation) Execute() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		var lastErr error
		
		for attempt := 1; attempt <= ro.strategy.maxRetries; attempt++ {
			// Execute the operation
			err := ro.operation()
			if err == nil {
				return nil // Success
			}

			lastErr = err

			// Check if we should retry
			if !ro.strategy.ShouldRetry(err, attempt) {
				break
			}

			// Call retry callback if provided
			if ro.onRetry != nil {
				ro.onRetry(attempt, err)
			}

			// Calculate delay and wait
			if attempt < ro.strategy.maxRetries {
				delay := ro.strategy.CalculateDelay(attempt)
				ro.model.logger.Debug("Retrying operation", 
					"attempt", attempt,
					"delay", delay,
					"error", err)
				
				time.Sleep(delay)
			}
		}

		// All retries exhausted, return the last error
		return NewErrorHandler(ro.model).HandleError(lastErr)
	})
}

// GracefulDegradation handles service degradation scenarios
type GracefulDegradation struct {
	model *Model
}

// NewGracefulDegradation creates a new graceful degradation handler
func NewGracefulDegradation(model *Model) *GracefulDegradation {
	return &GracefulDegradation{model: model}
}

// HandleAPIUnavailable handles API unavailability
func (gd *GracefulDegradation) HandleAPIUnavailable() tea.Cmd {
	// Switch to offline mode or cached responses
	return func() tea.Msg {
		return statusMsg{"API unavailable - some features may be limited", 5 * time.Second}
	}
}

// HandleRateLimitExceeded handles rate limit scenarios
func (gd *GracefulDegradation) HandleRateLimitExceeded(retryAfter time.Duration) tea.Cmd {
	// Queue requests or suggest alternative actions
	return func() tea.Msg {
		return statusMsg{fmt.Sprintf("Rate limited - please wait %v", retryAfter), 5 * time.Second}
	}
}

// HandleStorageUnavailable handles storage unavailability
func (gd *GracefulDegradation) HandleStorageUnavailable() tea.Cmd {
	// Use in-memory storage temporarily
	return func() tea.Msg {
		return statusMsg{"Storage unavailable - changes may not be saved", 5 * time.Second}
	}
}

// RecoveryManager manages error recovery
type RecoveryManager struct {
	model *Model
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(model *Model) *RecoveryManager {
	return &RecoveryManager{model: model}
}

// AttemptRecovery attempts to recover from an error
func (rm *RecoveryManager) AttemptRecovery(appErr *AppError) tea.Cmd {
	switch appErr.Type {
	case ErrorTypeNetwork:
		return rm.recoverFromNetworkError(appErr)
	case ErrorTypeAuth:
		return rm.recoverFromAuthError(appErr)
	case ErrorTypeStorage:
		return rm.recoverFromStorageError(appErr)
	case ErrorTypeRateLimit:
		return rm.recoverFromRateLimitError(appErr)
	default:
		return rm.defaultRecovery(appErr)
	}
}

// recoverFromNetworkError attempts to recover from network errors
func (rm *RecoveryManager) recoverFromNetworkError(appErr *AppError) tea.Cmd {
	// Could implement network connectivity check and retry
	return func() tea.Msg {
		return statusMsg{"Checking network connectivity...", 3 * time.Second}
	}
}

// recoverFromAuthError attempts to recover from authentication errors
func (rm *RecoveryManager) recoverFromAuthError(appErr *AppError) tea.Cmd {
	// Could prompt user to re-enter API key
	return func() tea.Msg {
		return statusMsg{"Please check your API credentials", 5 * time.Second}
	}
}

// recoverFromStorageError attempts to recover from storage errors
func (rm *RecoveryManager) recoverFromStorageError(appErr *AppError) tea.Cmd {
	// Could reinitialize storage or use fallback
	return func() tea.Msg {
		return statusMsg{"Attempting to recover storage...", 3 * time.Second}
	}
}

// recoverFromRateLimitError attempts to recover from rate limit errors
func (rm *RecoveryManager) recoverFromRateLimitError(appErr *AppError) tea.Cmd {
	if appErr.RetryAfter > 0 {
		return tea.Tick(appErr.RetryAfter, func(t time.Time) tea.Msg {
			return statusMsg{"Rate limit recovered - ready to continue", 2 * time.Second}
		})
	}
	return nil
}

// defaultRecovery provides default recovery behavior
func (rm *RecoveryManager) defaultRecovery(appErr *AppError) tea.Cmd {
	if !appErr.Recoverable {
		return nil
	}

	// Default retry after a short delay
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return statusMsg{"Ready to retry", 2 * time.Second}
	})
}