package marketaux_test

import (
	"testing"

	marketaux "github.com/tigusigalpa/marketaux-go"
)

func TestVersion(t *testing.T) {
	t.Parallel()
	if marketaux.Version == "" {
		t.Fatal("Version must not be empty")
	}
}
