package docx

import (
	"os"
	"testing"
)

var (
	testFile      = "./test/test.xml"
	totalRunCount = 7
	emptyRunCount = 2
	expectedTexts = []string{"TEXT0", "TEXT1", "TEXT2", "TEXT3", "TEXT4"}
)

func TestRunParser_FindRuns(t *testing.T) {
	docBytes := readFile(t, testFile)

	sut := NewRunParser(docBytes)
	err := sut.findRuns()
	if err != nil {
		t.Errorf("parser.findRuns failed: %s", err)
	}

	is := len(sut.Runs())
	if is != totalRunCount {
		t.Errorf("parser returned %d runs, expected %d", is, totalRunCount)
	}
	t.Logf("parser returned %d runs, expected %d", is, totalRunCount)
}

func TestRunParser_FindTextRuns(t *testing.T) {
	docBytes := readFile(t, testFile)

	sut := NewRunParser(docBytes)
	err := sut.findRuns()
	if err != nil {
		t.Errorf("parser.findRuns failed: %s", err)
	}
	err = sut.findTextRuns()
	if err != nil {
		t.Errorf("parser.findTextRuns failed: %s", err)
	}
}

func TestRun_GetText(t *testing.T) {
	docBytes := readFile(t, testFile)
	sut := NewRunParser(docBytes)
	err := sut.Execute()
	if err != nil {
		t.Errorf("parser.Execute failed: %s", err)
	}

	for _, expectedText := range expectedTexts {
		found := false
		for _, run := range sut.Runs().WithText() {
			text := run.GetText(docBytes)
			if text == expectedText {
				found = true
				t.Logf("found expected text %s", expectedText)
				continue
			}
		}
		if !found {
			t.Errorf("did not find expected text %s", expectedText)
		}
	}
}

func TestRun_WithText(t *testing.T) {
	docBytes := readFile(t, testFile)

	sut := NewRunParser(docBytes)
	err := sut.Execute()
	if err != nil {
		t.Errorf("parser.findRuns failed: %s", err)
	}

	is := len(sut.Runs().WithText())
	exp := totalRunCount - emptyRunCount
	if is != exp {
		t.Errorf("parser returned %d runs with text, expected %d", is, exp)
	}
}

func readFile(t testing.TB, path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		t.Error(err)
	}
	b := readBytes(f)
	n := len(b)
	if n == 0 {
		t.Errorf("nothing was read from test file %s", path)
	}

	return b
}
