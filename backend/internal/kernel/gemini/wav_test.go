package gemini

import (
	"encoding/binary"
	"testing"
)

// The WAV header, because a wrong number in it produces a file that plays as
// noise rather than one that fails to open.

func TestPCMRateFromMimeReadsTheRate(t *testing.T) {
	cases := map[string]int{
		"audio/L16;codec=pcm;rate=24000":  24000,
		"audio/L16; codec=pcm; rate=8000": 8000,
		"audio/L16;codec=pcm":             defaultPCMRate, // not stated
		"":                                defaultPCMRate,
		"audio/L16;rate=nonsense":         defaultPCMRate,
		"audio/L16;rate=-1":               defaultPCMRate,
	}
	for mime, want := range cases {
		if got := PCMRateFromMime(mime); got != want {
			t.Errorf("PCMRateFromMime(%q) = %d, want %d", mime, got, want)
		}
	}
}

func TestPCMToWAVWritesAHeaderThatDescribesTheAudio(t *testing.T) {
	pcm := make([]byte, 960) // 20ms of 16-bit mono at 24kHz
	wav := PCMToWAV(pcm, 24000)

	if len(wav) != 44+len(pcm) {
		t.Fatalf("the file is %d bytes, want %d", len(wav), 44+len(pcm))
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" || string(wav[36:40]) != "data" {
		t.Fatalf("the chunk names are wrong: %q", wav[:44])
	}
	if got := binary.LittleEndian.Uint32(wav[4:8]); got != uint32(36+len(pcm)) {
		t.Errorf("the RIFF size is %d", got)
	}
	if got := binary.LittleEndian.Uint32(wav[24:28]); got != 24000 {
		t.Errorf("the sample rate is %d", got)
	}
	// Byte rate = rate × channels × bits/8, and a player that disagrees with it
	// plays the audio at the wrong speed.
	if got := binary.LittleEndian.Uint32(wav[28:32]); got != 24000*2 {
		t.Errorf("the byte rate is %d, want %d", got, 24000*2)
	}
	if got := binary.LittleEndian.Uint16(wav[32:34]); got != 2 {
		t.Errorf("the block alignment is %d", got)
	}
	if got := binary.LittleEndian.Uint32(wav[40:44]); got != uint32(len(pcm)) {
		t.Errorf("the data size is %d", got)
	}
}

// A rate of zero is what an unparsed mime type used to yield, and dividing by
// it would have produced a header no player accepts.
func TestPCMToWAVFallsBackToTheDefaultRate(t *testing.T) {
	wav := PCMToWAV(make([]byte, 32), 0)
	if got := binary.LittleEndian.Uint32(wav[24:28]); got != defaultPCMRate {
		t.Errorf("the sample rate is %d, want the default %d", got, defaultPCMRate)
	}
}
