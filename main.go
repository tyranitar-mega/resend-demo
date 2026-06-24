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

	email := os.Getenv("RESEND_EMAIL")
	if email == "" {
		panic("RESEND_EMAIL environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := resend.NewClient(apiKey)
	response, err := client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    fmt.Sprintf("Tyranitar Mega <noreply@%s>", domain),
		To:      []string{email},
		Subject: "Test Email from Resend",
		Html:    "<h1>Welcome!</h1><p>This is an <strong>HTML</strong> email.</p>",
	})
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return
	}
	fmt.Printf("Email sent successfully! Response: %+v\n", response)
}
