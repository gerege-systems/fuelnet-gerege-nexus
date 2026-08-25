/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package gemini

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"
)

// Speech comes back as raw PCM — typically `audio/L16;codec=pcm;rate=24000` —
// and no browser will play that. These two turn it into something an <audio>
// element accepts, which is forty-four bytes of header and nothing else.

const defaultPCMRate = 24000

// PCMRateFromMime reads the sample rate out of the media type, falling back to
// the rate Gemini's speech models use.
func PCMRateFromMime(mime string) int {
	for _, parameter := range strings.Split(mime, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
		if !ok || !strings.EqualFold(key, "rate") {
			continue
		}
		if rate, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && rate > 0 {
			return rate
		}
	}
	return defaultPCMRate
}

// PCMToWAV wraps 16-bit mono PCM in a WAV container.
func PCMToWAV(pcm []byte, sampleRate int) []byte {
	const (
		channels      = 1
		bitsPerSample = 16
	)
	if sampleRate <= 0 {
		sampleRate = defaultPCMRate
	}
	// Every size below is a 32-bit field, and every value put in one is
	// bounded long before it arrives: the audio comes from a response capped at
	// 32 MiB, and the rate is a few tens of thousands. The conversions are
	// stated rather than suppressed — a header written from a length that did
	// not fit would be a file that plays as noise.
	if len(pcm) > math.MaxUint32-44 || sampleRate > math.MaxUint32/4 {
		return nil
	}
	// #nosec G115 -- bounded by the check above: the length fits a 32-bit field
	pcmLen := uint32(len(pcm))
	// #nosec G115 -- bounded by the check above: the rate fits a 32-bit field
	rate := uint32(sampleRate)

	out := make([]byte, 0, 44+len(pcm))
	u32 := func(v uint32) []byte { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); return b }
	u16 := func(v uint16) []byte { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); return b }

	out = append(out, []byte("RIFF")...)
	out = append(out, u32(36+pcmLen)...)
	out = append(out, []byte("WAVE")...)
	out = append(out, []byte("fmt ")...)
	out = append(out, u32(16)...) // the size of this chunk
	out = append(out, u16(1)...)  // 1 = uncompressed PCM
	out = append(out, u16(channels)...)
	out = append(out, u32(rate)...)
	out = append(out, u32(rate*channels*bitsPerSample/8)...) // the byte rate
	out = append(out, u16(channels*bitsPerSample/8)...)      // the block alignment
	out = append(out, u16(bitsPerSample)...)
	out = append(out, []byte("data")...)
	out = append(out, u32(pcmLen)...)
	out = append(out, pcm...)
	return out
}
