/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Package gemini is this platform's REST client for Google's Gemini API.
//
// It replaces open-gerege-core/pkg/gemini, and for the reason eidrp replaced
// that library's eID client: a dependency released from another repository on
// another schedule sits underneath something this platform's users touch every
// day, and the half of it this platform uses is small enough to own.
//
// Small enough is meant literally. What is here is the generateContent call and
// the types it takes; the library's embedding client, its streaming surface and
// its model-fallback bookkeeping are not, because nothing in this repository
// asked for them.
//
//	POST {base}/models/{model}:generateContent
//	Auth: x-goog-api-key: <key>
//
// Transient failures — a dropped connection, a 429, a 5xx — are retried three
// times with exponential backoff and then reported as ErrUnavailable, which is
// the signal a caller needs to answer with something other than an error.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNotConfigured is a deployment with no API key. The platform boots and
// serves everything else; what it does not do is pretend the copilot is there.
var ErrNotConfigured = errors.New("gemini: no API key is configured")

// ErrUnavailable is a *temporary* failure: the network, a rate limit, or a
// server error that outlasted every retry. Callers answer it differently from a
// permanent one — a fallback reply rather than a broken feature.
var ErrUnavailable = errors.New("gemini: service unavailable")

const (
	defaultBase  = "https://generativelanguage.googleapis.com/v1beta"
	defaultModel = "gemini-2.5-flash"
	// maxRespBytes bounds what is read into memory. A speech answer embeds
	// base64 PCM inside the JSON, so a long one is several MiB; 4 MiB truncated
	// them and the decode failed with a message about JSON rather than about
	// size.
	maxRespBytes = 32 << 20

	// One attempt and two retries, at 500ms then 1s.
	maxAttempts    = 3
	initialBackoff = 500 * time.Millisecond
	httpTimeout    = 60 * time.Second
)

// Part is one piece of a turn: text, a function call the model decided on, the
// result of one this platform ran, or inline media.
type Part struct {
	Text             string            `json:"text,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
	InlineData       *Blob             `json:"inlineData,omitempty"`
}

// Blob is inline media: base64 bytes and their type. Audio in, speech out.
type Blob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// FunctionCall is the model asking for a function to be run.
type FunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// FunctionResponse hands the result back to the model.
type FunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// Content is one turn. Role is "user" or "model".
type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}

// FunctionDeclaration is a function offered to the model. Parameters is JSON
// Schema, left as a map because the caller declares its own shape.
type FunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// Tool is the set of functions declared for one request.
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations"`
}

// GenerationConfig is the optional half of a request. ResponseModalities and
// SpeechConfig are what turn a request into speech.
type GenerationConfig struct {
	Temperature        *float64      `json:"temperature,omitempty"`
	TopP               *float64      `json:"topP,omitempty"`
	MaxOutputTokens    int           `json:"maxOutputTokens,omitempty"`
	ResponseModalities []string      `json:"responseModalities,omitempty"`
	SpeechConfig       *SpeechConfig `json:"speechConfig,omitempty"`
	// ResponseMimeType of "application/json" makes the model answer with JSON
	// and nothing else — no prose, no code fence to strip.
	ResponseMimeType string `json:"responseMimeType,omitempty"`
	ResponseSchema   any    `json:"responseSchema,omitempty"`
}

// SpeechConfig chooses the voice.
type SpeechConfig struct {
	VoiceConfig *VoiceConfig `json:"voiceConfig,omitempty"`
}

type VoiceConfig struct {
	PrebuiltVoiceConfig *PrebuiltVoiceConfig `json:"prebuiltVoiceConfig,omitempty"`
}

type PrebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

// Request is the generateContent body.
type Request struct {
	SystemInstruction *Content          `json:"systemInstruction,omitempty"`
	Contents          []Content         `json:"contents"`
	Tools             []Tool            `json:"tools,omitempty"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
}

// Candidate is one answer.
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason,omitempty"`
}

// Response is what came back.
type Response struct {
	Candidates []Candidate `json:"candidates"`
}

// Text is every text part of the first candidate, joined.
func (r Response) Text() string {
	if len(r.Candidates) == 0 {
		return ""
	}
	var out strings.Builder
	for _, part := range r.Candidates[0].Content.Parts {
		out.WriteString(part.Text)
	}
	return strings.TrimSpace(out.String())
}

// FunctionCalls is every function the first candidate asked for. Empty means
// the model answered in words.
func (r Response) FunctionCalls() []FunctionCall {
	if len(r.Candidates) == 0 {
		return nil
	}
	var calls []FunctionCall
	for _, part := range r.Candidates[0].Content.Parts {
		if part.FunctionCall != nil {
			calls = append(calls, *part.FunctionCall)
		}
	}
	return calls
}

// InlineAudio is the first audio part of the first candidate, or nil.
func (r Response) InlineAudio() *Blob {
	if len(r.Candidates) == 0 {
		return nil
	}
	for _, part := range r.Candidates[0].Content.Parts {
		if part.InlineData != nil && strings.HasPrefix(part.InlineData.MimeType, "audio/") {
			return part.InlineData
		}
	}
	return nil
}

// ModelContent is the first candidate's turn, for appending to a conversation
// during a function-calling loop.
func (r Response) ModelContent() Content {
	if len(r.Candidates) == 0 {
		return Content{Role: "model"}
	}
	content := r.Candidates[0].Content
	if content.Role == "" {
		content.Role = "model"
	}
	return content
}

// Generator is the call itself, as an interface, so a test can answer it.
type Generator interface {
	GenerateContent(ctx context.Context, req Request) (Response, error)
}

// Client talks to the API.
type Client struct {
	base   string
	apiKey string
	model  string
	http   *http.Client
	// sleep is replaced in tests, so a retry does not cost a second.
	sleep func(ctx context.Context, d time.Duration) error
}

// NewClient builds a client. Empty base and model take the defaults; an empty
// key makes every call answer ErrNotConfigured rather than fail at the wire.
func NewClient(base, apiKey, model string) *Client {
	if base = strings.TrimRight(strings.TrimSpace(base), "/"); base == "" {
		base = defaultBase
	}
	if model = strings.TrimSpace(model); model == "" {
		model = defaultModel
	}
	return &Client{
		base: base, apiKey: apiKey, model: model,
		http:  &http.Client{Timeout: httpTimeout},
		sleep: sleepCtx,
	}
}

// GenerateContent calls the API, retrying what is worth retrying.
func (c *Client) GenerateContent(ctx context.Context, req Request) (Response, error) {
	if c.apiKey == "" {
		return Response{}, ErrNotConfigured
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if err := c.sleep(ctx, initialBackoff<<(attempt-1)); err != nil {
				return Response{}, fmt.Errorf("gemini: waiting to retry: %w", err)
			}
		}
		resp, retryable, err := c.generateOnce(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !retryable {
			return Response{}, err
		}
	}
	return Response{}, fmt.Errorf("%w: %d attempts failed: %w", ErrUnavailable, maxAttempts, lastErr)
}

// generateOnce is one attempt. retryable says whether trying again could help.
func (c *Client) generateOnce(ctx context.Context, req Request) (Response, bool, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return Response{}, false, fmt.Errorf("gemini: encode the request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent", c.base, c.model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return Response{}, false, fmt.Errorf("gemini: build the request: %w", err)
	}
	httpReq.Header.Set("x-goog-api-key", c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		// A cancelled context is not going to go better on the next attempt.
		if ctx.Err() != nil {
			return Response{}, false, fmt.Errorf("gemini: http: %w", err)
		}
		return Response{}, true, fmt.Errorf("%w: http: %w", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
	if readErr != nil {
		// A body that stopped mid-read is a hiccup, not a broken request: half
		// a JSON document is not something to report as a permanent failure.
		if ctx.Err() != nil {
			return Response{}, false, fmt.Errorf("gemini: read the answer: %w", readErr)
		}
		return Response{}, true, fmt.Errorf("gemini: read the answer: %w", readErr)
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return Response{}, true, fmt.Errorf("gemini: status %d: %s", resp.StatusCode, snippet(raw))
	case resp.StatusCode >= 300:
		// A bad request or a rejected key: retrying changes nothing.
		return Response{}, false, fmt.Errorf("gemini: status %d: %s", resp.StatusCode, snippet(raw))
	}

	// At the cap the body is probably cut short, and the next attempt would be
	// cut short in the same place. Say so, rather than reporting a decode error
	// about a document that was fine until it was truncated.
	if int64(len(raw)) >= maxRespBytes {
		return Response{}, false,
			fmt.Errorf("gemini: the answer exceeded %d bytes and was truncated", int64(maxRespBytes))
	}

	var out Response
	if err := json.Unmarshal(raw, &out); err != nil {
		return Response{}, false, fmt.Errorf("gemini: decode the answer: %w", err)
	}
	return out, false, nil
}

// sleepCtx waits, but not past the end of the request.
func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if runes := []rune(s); len(runes) > 200 {
		return string(runes[:200])
	}
	return s
}
