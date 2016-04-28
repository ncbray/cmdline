package cmdline

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHelp(t *testing.T) {
	app := MakeApp("foo")
	var b bytes.Buffer
	app.WriteHelp(&b)
	assert.Equal(t, "usage: foo\n", b.String())
}
