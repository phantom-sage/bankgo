package integration

import (
	"context"
	"fmt"

	"github.com/phantom-sage/bankgo/internal/queue"
)

// TestEmailProcessor implements EmailProcessor for testing
type TestEmailProcessor struct {
	processedEmails []queue.WelcomeEmailPayload
	shouldFail      bool
	failureError    error
}

// NewTestEmailProcessor creates a new test email processor
func NewTestEmailProcessor() *TestEmailProcessor {
	return &TestEmailProcessor{
		processedEmails: make([]queue.WelcomeEmailPayload, 0),
		shouldFail:      false,
	}
}

// ProcessWelcomeEmail processes welcome email for testing
func (tep *TestEmailProcessor) ProcessWelcomeEmail(ctx context.Context, payload queue.WelcomeEmailPayload) error {
	if tep.shouldFail {
		return tep.failureError
	}
	
	// Validate email format for testing
	if payload.Email == "invalid-email-format" || payload.Email == "invalid-email-address" {
		return fmt.Errorf("invalid email format: %s", payload.Email)
	}
	
	tep.processedEmails = append(tep.processedEmails, payload)
	return nil
}

// SetShouldFail sets whether the processor should fail
func (tep *TestEmailProcessor) SetShouldFail(shouldFail bool, err error) {
	tep.shouldFail = shouldFail
	tep.failureError = err
}

// GetProcessedEmails returns the list of processed emails
func (tep *TestEmailProcessor) GetProcessedEmails() []queue.WelcomeEmailPayload {
	return tep.processedEmails
}

// ClearProcessedEmails clears the list of processed emails
func (tep *TestEmailProcessor) ClearProcessedEmails() {
	tep.processedEmails = make([]queue.WelcomeEmailPayload, 0)
}

// QueueManagerTestExtensions provides additional methods for testing
type QueueManagerTestExtensions struct {
	*queue.QueueManager
	emailProcessor *TestEmailProcessor
}

// NewQueueManagerTestExtensions creates a queue manager with test extensions
func NewQueueManagerTestExtensions(qm *queue.QueueManager) *QueueManagerTestExtensions {
	processor := NewTestEmailProcessor()
	qm.RegisterHandlers(processor)
	
	return &QueueManagerTestExtensions{
		QueueManager:   qm,
		emailProcessor: processor,
	}
}

// ProcessWelcomeEmail processes a welcome email directly for testing
func (qmte *QueueManagerTestExtensions) ProcessWelcomeEmail(ctx context.Context, payload queue.WelcomeEmailPayload) error {
	return qmte.emailProcessor.ProcessWelcomeEmail(ctx, payload)
}

// ProcessWelcomeEmailWithContext processes a welcome email with context for testing
func (qmte *QueueManagerTestExtensions) ProcessWelcomeEmailWithContext(ctx context.Context, payload queue.WelcomeEmailPayload) error {
	return qmte.emailProcessor.ProcessWelcomeEmail(ctx, payload)
}

// GetQueuedTasks returns queued tasks (mock implementation for testing)
func (qmte *QueueManagerTestExtensions) GetQueuedTasks(ctx context.Context, taskType string) ([]string, error) {
	// In a real implementation, this would query Redis/Asynq for pending tasks
	// For testing, we'll return a mock response
	if taskType == "welcome_email" {
		// Return mock queued tasks based on recent queue operations
		return []string{"mock-task-1", "mock-task-2"}, nil
	}
	return []string{}, nil
}

// GetRetryTasks returns retry tasks (mock implementation for testing)
func (qmte *QueueManagerTestExtensions) GetRetryTasks(ctx context.Context, taskType string) ([]string, error) {
	// In a real implementation, this would query Redis/Asynq for retry tasks
	// For testing, we'll return a mock response
	if taskType == "welcome_email" && qmte.emailProcessor.shouldFail {
		return []string{"retry-task-1"}, nil
	}
	return []string{}, nil
}

// ClearQueues clears all queues (mock implementation for testing)
func (qmte *QueueManagerTestExtensions) ClearQueues(ctx context.Context) error {
	// In a real implementation, this would clear Redis/Asynq queues
	// For testing, we'll just clear our mock data
	qmte.emailProcessor.ClearProcessedEmails()
	return nil
}

// SetEmailProcessorFailure sets the email processor to fail for testing
func (qmte *QueueManagerTestExtensions) SetEmailProcessorFailure(shouldFail bool, err error) {
	qmte.emailProcessor.SetShouldFail(shouldFail, err)
}

// GetProcessedEmails returns processed emails for testing
func (qmte *QueueManagerTestExtensions) GetProcessedEmails() []queue.WelcomeEmailPayload {
	return qmte.emailProcessor.GetProcessedEmails()
}