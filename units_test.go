package spacehoarder

import (
	"testing"
)

func TestFancySize(t *testing.T) {

	if s := FancySize(125); s != "125.0B" {
		t.Fatal("Format wrong: ", s)
	}

	if s := FancySize(1024); s != "1.0KB" {
		t.Fatal("Format wrong: ", s)
	}

	if s := FancySize(1024*2 + 1024/2); s != "2.5KB" {
		t.Fatal("Format wrong: ", s)
	}

}
