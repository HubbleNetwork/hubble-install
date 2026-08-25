package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	_ = r.Close()
	return buf.String()
}

func TestPrintCompletionBannerSatellite(t *testing.T) {
	out := captureStdout(t, func() {
		PrintCompletionBanner(time.Second, "org", "token", "vivavious-vampire-bat", true)
	})

	want := []string{
		`Your device "vivavious-vampire-bat" is now broadcasting on the Hubble Satellite Network.`,
		"Your device gets 1+ satellite passes per day. Place your device outdoors with a clear line-of-sight to the sky to transmit at the next opportunity.",
		"Return to https://dash.hubble.com to view upcoming satellite passes.",
	}
	for _, s := range want {
		if !strings.Contains(out, s) {
			t.Fatalf("satellite banner missing %q\noutput:\n%s", s, out)
		}
	}

	forbid := []string{
		"Hubble Terrestrial Network",
		"Hubble Connect mobile app",
		"capture device packets",
	}
	for _, s := range forbid {
		if strings.Contains(out, s) {
			t.Fatalf("satellite banner unexpectedly contains %q\noutput:\n%s", s, out)
		}
	}
}

func TestPrintCompletionBannerTerrestrial(t *testing.T) {
	out := captureStdout(t, func() {
		PrintCompletionBanner(time.Second, "org", "token", "vivavious-vampire-bat", false)
	})

	want := []string{
		`Your device "vivavious-vampire-bat" is now broadcasting on the Hubble Terrestrial Network`,
		"Hubble Connect mobile app",
		"Return to https://dash.hubble.com to capture device packets!",
	}
	for _, s := range want {
		if !strings.Contains(out, s) {
			t.Fatalf("terrestrial banner missing %q\noutput:\n%s", s, out)
		}
	}

	if strings.Contains(out, "Hubble Satellite Network") {
		t.Fatalf("terrestrial banner unexpectedly contains satellite copy\noutput:\n%s", out)
	}
}
