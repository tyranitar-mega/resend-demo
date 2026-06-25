// Copyright 2026 tyranitar-mega
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/resend/resend-go/v3"
	"github.com/tyranitar-meta/resend-demo/zerodrop"
)

func main() {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		panic("RESEND_API_KEY environment variable is not set")
	}

	domain := os.Getenv("RESEND_DOMAIN")
	if domain == "" {
		panic("RESEND_DOMAIN environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	z := zerodrop.NewClient()

	inbox := z.GenerateInbox()
	fmt.Printf("[ZeroDrop] Generated inbox: %s\n", inbox)

	otp := "123456"
	verifyLink := "https://example.com/verify?token=abc123def456"
	htmlBody := fmt.Sprintf(`<h1>Verify your email</h1>
<p>Your verification code is: <strong>%s</strong></p>
<p>Or click this link to verify: <a href="%s">%s</a></p>
<p>This code expires in 10 minutes.</p>`, otp, verifyLink, verifyLink)

	client := resend.NewClient(apiKey)
	_, err := client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    fmt.Sprintf("Tyranitar Mega <noreply@%s>", domain),
		To:      []string{inbox},
		Subject: "Verify your email address",
		Html:    htmlBody,
	})
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return
	}
	fmt.Printf("[Resend] Verification email sent to %s\n", inbox)

	fmt.Println("[ZeroDrop] Waiting for email to arrive...")
	email, err := z.WaitForLatest(ctx, inbox, &zerodrop.WaitOptions{
		Timeout:      30 * time.Second,
		PollInterval: 2 * time.Second,
	})
	if err != nil {
		fmt.Printf("Error waiting for email: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("=== Email Received ===")
	fmt.Printf("ID:        %s\n", email.ID)
	fmt.Printf("From:      %s\n", email.From)
	fmt.Printf("To:        %s\n", email.To)
	fmt.Printf("Subject:   %s\n", email.Subject)
	fmt.Printf("Received:  %s\n", email.ReceivedAt.Format(time.RFC3339))
	fmt.Println("--- Body ---")
	fmt.Println(email.Body)
	fmt.Println("--- Auto-extracted ---")
	if email.OTP != nil {
		fmt.Printf("OTP:       %s\n", *email.OTP)
	} else {
		fmt.Println("OTP:       (not detected)")
	}
	if email.MagicLink != nil {
		fmt.Printf("MagicLink: %s\n", *email.MagicLink)
	} else {
		fmt.Println("MagicLink: (not detected)")
	}

	if email.OTP != nil && *email.OTP == otp {
		fmt.Println("\nOTP verification PASSED!")
	} else if email.OTP != nil {
		fmt.Printf("\nOTP mismatch: expected %s, got %s\n", otp, *email.OTP)
	}
}
