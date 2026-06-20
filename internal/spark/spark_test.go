package spark

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSparkArtNotInStdout(t *testing.T) {
	resetSparkForTest(t)

	var stderr bytes.Buffer
	SetOutput(&stderr)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	Greet()

	_ = w.Close()
	os.Stdout = oldStdout
	stdout, _ := io.ReadAll(r)

	if len(stdout) != 0 {
		t.Fatalf("spark wrote to stdout: %q", string(stdout))
	}
	if !strings.Contains(stderr.String(), "Spark is watching your spend") {
		t.Fatalf("expected Spark output on stderr, got %q", stderr.String())
	}
}

func TestSparkRespectsQuiet(t *testing.T) {
	resetSparkForTest(t)

	var stderr bytes.Buffer
	SetOutput(&stderr)
	t.Setenv("FUSE_QUIET", "1")

	Greet()
	SoftCapWarning(52, 50, "daily")
	HardCapBlocked(3.2, 98.5, 100, "daily")

	if stderr.Len() != 0 {
		t.Fatalf("expected quiet Spark output, got %q", stderr.String())
	}
}

func TestSparkColorStripsWhenNotTTY(t *testing.T) {
	resetSparkForTest(t)

	got := Colorize("Spark", Green)
	if strings.Contains(got, "\033[") {
		t.Fatalf("expected no ANSI codes when not a tty, got %q", got)
	}
}

func TestSparkHardCapMessageFormat(t *testing.T) {
	resetSparkForTest(t)

	var stderr bytes.Buffer
	SetOutput(&stderr)
	HardCapBlocked(3.2, 98.5, 100, "daily")

	out := stderr.String()
	for _, want := range []string{"SPARK: HARD CAP TRIGGERED", "$3.20", "$98.50", "$100.00"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got %q", want, out)
		}
	}
}

func TestSparkNoCheerfulWords(t *testing.T) {
	resetSparkForTest(t)

	var stderr bytes.Buffer
	SetOutput(&stderr)
	Greet()
	ProxyStarted("localhost:8787")
	SoftCapWarning(52, 50, "daily")
	HardCapBlocked(3.2, 98.5, 100, "daily")
	ConfigLoaded()
	ConfigInvalid(os.ErrInvalid)
	HistoryEmpty()
	Shutdown()
	Tip()

	out := stderr.String()
	for _, forbidden := range []string{"Great", "Awesome", "🎉"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("forbidden cheerful word %q found in %q", forbidden, out)
		}
	}
}

func resetSparkForTest(t *testing.T) {
	t.Helper()
	SetOutput(io.Discard)
	SetQuiet(false)
	SetNoMascot(false)
	t.Setenv("FUSE_QUIET", "")
	t.Setenv("FUSE_NO_MASCOT", "")
}
