package godanticlint

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	testdata := filepath.Join(wd, "testdata")
	analysistest.Run(t, testdata, Analyzer, "testdata/src/valid", "testdata/src/invalid")
}
