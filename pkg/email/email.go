package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/phantom-sage/bankgo/internal/queue"
)

// Service represents the email service
type Service struct {
	config config.EmailConfig
	auth   smtp.Auth
}

// NewService creates a new email service
func NewService(cfg config.EmailConfig) *Service {
	auth := smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	
	return &Service{
		config: cfg,
		auth:   auth,
	}
}

// WelcomeEmailData represents the data for welcome email template
type WelcomeEmailData struct {
	FirstName string
	LastName  string
	Email     string
}

// welcomeEmailTemplate is the HTML template for welcome emails
const welcomeEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to Bank API</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background-color: #4CAF50;
            color: white;
            padding: 20px;
            text-align: center;
            border-radius: 5px 5px 0 0;
        }
        .content {
            background-color: #f9f9f9;
            padding: 30px;
            border-radius: 0 0 5px 5px;
        }
        .footer {
            text-align: center;
            margin-top: 20px;
            font-size: 12px;
            color: #666;
        }
        .button {
            display: inline-block;
            background-color: #4CAF50;
            color: white;
            padding: 12px 24px;
            text-decoration: none;
            border-radius: 4px;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Welcome to Bank API!</h1>
    </div>
    <div class="content">
        <h2>Hello {{.FirstName}} {{.LastName}},</h2>
        <p>Welcome to our banking service! We're excited to have you on board.</p>
        
        <p>Your account has been successfully created with the email address: <strong>{{.Email}}</strong></p>
        
        <p>With our banking API, you can:</p>
        <ul>
            <li>Create multiple accounts with different currencies</li>
            <li>Transfer money between accounts securely</li>
            <li>View your account balances and transaction history</li>
            <li>Manage your financial portfolio efficiently</li>
        </ul>
        
        <p>If you have any questions or need assistance, please don't hesitate to contact our support team.</p>
        
        <p>Thank you for choosing our banking service!</p>
        
        <p>Best regards,<br>
        The Bank API Team</p>
    </div>
    <div class="footer">
        <p>This is an automated message. Please do not reply to this email.</p>
    </div>
</body>
</html>
`

// SendWelcomeEmail sends a welcome email to a new user
func (s *Service) SendWelcomeEmail(ctx context.Context, data WelcomeEmailData) error {
	// Parse the email template
	tmpl, err := template.New("welcome").Parse(welcomeEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse welcome email template: %w", err)
	}

	// Execute the template with data
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute welcome email template: %w", err)
	}

	// Prepare email message
	subject := "Welcome to Bank API - Your Account is Ready!"
	msg := s.buildEmailMessage(data.Email, subject, body.String())

	// Send email
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	to := []string{data.Email}
	
	if err := smtp.SendMail(addr, s.auth, s.config.FromEmail, to, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send welcome email to %s: %w", data.Email, err)
	}

	return nil
}

// buildEmailMessage builds the complete email message with headers
func (s *Service) buildEmailMessage(to, subject, body string) string {
	var msg strings.Builder
	
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromEmail))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)
	
	return msg.String()
}

// ProcessWelcomeEmail processes a welcome email task from the queue
func (s *Service) ProcessWelcomeEmail(ctx context.Context, payload queue.WelcomeEmailPayload) error {
	startTime := time.Now()
	
	// Get correlation ID from context
	correlationID := ""
	if id := ctx.Value("correlation_id"); id != nil {
		if cid, ok := id.(string); ok {
			correlationID = cid
		}
	}
	
	// Create logger for this operation (will be replaced with proper logger in future tasks)
	logger := log.New(os.Stdout, "[EMAIL] ", log.LstdFlags)
	
	logger.Printf("Processing welcome email - User ID: %d, Email: %s, Correlation ID: %s", 
		payload.UserID, payload.Email, correlationID)
	data := WelcomeEmailData{
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Email:     payload.Email,
	}

	if err := s.SendWelcomeEmail(ctx, data); err != nil {
		duration := time.Since(startTime)
		logger.Printf("Failed to send welcome email - User ID: %d, Email: %s, Duration: %v, Error: %v, Correlation ID: %s", 
			payload.UserID, payload.Email, duration, err, correlationID)
		return fmt.Errorf("failed to process welcome email for user %d: %w", payload.UserID, err)
	}

	duration := time.Since(startTime)
	logger.Printf("Welcome email sent successfully - User ID: %d, Email: %s, Duration: %v, Correlation ID: %s", 
		payload.UserID, payload.Email, duration, correlationID)

	return nil
}

// QueueWelcomeEmail queues a welcome email task
func (s *Service) QueueWelcomeEmail(ctx context.Context, qm *queue.QueueManager, userID int, email, firstName, lastName string) error {
	payload := queue.WelcomeEmailPayload{
		UserID:    userID,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}

	return qm.QueueWelcomeEmail(ctx, payload)
}