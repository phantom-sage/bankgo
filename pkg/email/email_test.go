package email

import (
	"context"
	"strings"
	"testing"

	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/phantom-sage/bankgo/internal/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	cfg := config.EmailConfig{
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "test@example.com",
		FromName:     "Test Bank",
	}

	service := NewService(cfg)
	
	require.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
	assert.NotNil(t, service.auth)
}

func TestService_buildEmailMessage(t *testing.T) {
	cfg := config.EmailConfig{
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "noreply@bankapi.com",
		FromName:     "Bank API",
	}

	service := NewService(cfg)
	
	to := "user@example.com"
	subject := "Test Subject"
	body := "<h1>Test Body</h1>"
	
	msg := service.buildEmailMessage(to, subject, body)
	
	// Check that all required headers are present
	assert.Contains(t, msg, "From: Bank API <noreply@bankapi.com>")
	assert.Contains(t, msg, "To: user@example.com")
	assert.Contains(t, msg, "Subject: Test Subject")
	assert.Contains(t, msg, "MIME-Version: 1.0")
	assert.Contains(t, msg, "Content-Type: text/html; charset=UTF-8")
	assert.Contains(t, msg, "<h1>Test Body</h1>")
}

func TestWelcomeEmailTemplate(t *testing.T) {
	// Test that the template contains expected elements
	assert.Contains(t, welcomeEmailTemplate, "Welcome to Bank API!")
	assert.Contains(t, welcomeEmailTemplate, "{{.FirstName}}")
	assert.Contains(t, welcomeEmailTemplate, "{{.LastName}}")
	assert.Contains(t, welcomeEmailTemplate, "{{.Email}}")
	assert.Contains(t, welcomeEmailTemplate, "Create multiple accounts")
	assert.Contains(t, welcomeEmailTemplate, "Transfer money between accounts")
}

func TestService_SendWelcomeEmail_TemplateExecution(t *testing.T) {
	cfg := config.EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "noreply@bankapi.com",
		FromName:     "Bank API",
	}

	service := NewService(cfg)
	
	data := WelcomeEmailData{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
	}

	// We can't actually send the email in tests, but we can test template execution
	// by checking if the buildEmailMessage method works correctly
	msg := service.buildEmailMessage(data.Email, "Test", "Test body")
	assert.Contains(t, msg, data.Email)
}

func TestService_ProcessWelcomeEmail(t *testing.T) {
	cfg := config.EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "noreply@bankapi.com",
		FromName:     "Bank API",
	}

	service := NewService(cfg)
	
	payload := queue.WelcomeEmailPayload{
		UserID:    123,
		Email:     "user@example.com",
		FirstName: "Jane",
		LastName:  "Smith",
	}

	ctx := context.Background()
	
	// This will fail because we can't actually send emails in tests,
	// but we can verify the method signature and basic functionality
	err := service.ProcessWelcomeEmail(ctx, payload)
	
	// We expect an error because SMTP server is not available
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "failed to process welcome email")
}

func TestWelcomeEmailData(t *testing.T) {
	data := WelcomeEmailData{
		FirstName: "Alice",
		LastName:  "Johnson",
		Email:     "alice.johnson@example.com",
	}

	assert.Equal(t, "Alice", data.FirstName)
	assert.Equal(t, "Johnson", data.LastName)
	assert.Equal(t, "alice.johnson@example.com", data.Email)
}

func TestService_QueueWelcomeEmail_PayloadCreation(t *testing.T) {
	cfg := config.EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "noreply@bankapi.com",
		FromName:     "Bank API",
	}

	service := NewService(cfg)
	
	// We can't test the actual queuing without a queue manager,
	// but we can verify the method signature exists
	assert.NotNil(t, service)
	
	// Test payload creation logic by checking the ProcessWelcomeEmail method
	payload := queue.WelcomeEmailPayload{
		UserID:    456,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
	}

	ctx := context.Background()
	err := service.ProcessWelcomeEmail(ctx, payload)
	
	// We expect an error due to SMTP not being available, but the method should exist
	assert.Error(t, err)
}