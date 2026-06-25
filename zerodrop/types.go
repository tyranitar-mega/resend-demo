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

package zerodrop

import (
	"fmt"
	"time"
)

type Email struct {
	ID         string    `json:"id"`
	From       string    `json:"from"`
	To         string    `json:"to"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	RawBody    string    `json:"rawBody"`
	ReceivedAt time.Time `json:"receivedAt"`
	OTP        *string   `json:"otp"`
	MagicLink  *string   `json:"magicLink"`
}

type WaitOptions struct {
	Timeout      time.Duration
	PollInterval time.Duration
	SSE          *bool
}

type TimeoutError struct {
	Inbox   string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf(
		"ZeroDrop: No email received at %q within %v. Check that your app is sending to the correct address.",
		e.Inbox, e.Timeout,
	)
}

type AuthError struct{}

func (e *AuthError) Error() string {
	return "ZeroDrop: Invalid or missing API key."
}

type NetworkError struct {
	Message string
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf(
		"ZeroDrop: Network error — %s. Check https://zerodrop.instatus.com for service status.",
		e.Message,
	)
}
