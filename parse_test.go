package cmdline

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type mockFlag struct {
	long   string
	short  string
	hasArg bool
}

type mockParseObserver struct {
	all       []*mockFlag
	short     map[string]*mockFlag
	long      map[string]*mockFlag
	b         bytes.Buffer
	failAfter int
	banArgs   bool
	numErrors int
}

func (o *mockParseObserver) flag(long string, short string, hasArg bool) {
	f := &mockFlag{long: long, short: short, hasArg: hasArg}
	o.all = append(o.all, f)
	if long != "" {
		o.long[long] = f
	}
	if short != "" {
		o.short[short] = f
	}
}

func (o *mockParseObserver) longFlagInfo(name string) (bool, bool) {
	f, ok := o.long[name]
	if ok {
		return true, f.hasArg
	} else {
		return false, false
	}
}

func (o *mockParseObserver) shortFlagInfo(name rune) (bool, bool) {
	f, ok := o.short[string(name)]
	if ok {
		return true, f.hasArg
	} else {
		return false, false
	}
}

func (o *mockParseObserver) spaceIfNeeded() {
	if o.b.Len() > 0 {
		o.b.WriteString(" ")
	}
}

func (o *mockParseObserver) injectFault() bool {
	if o.failAfter == 0 {
		return false
	} else if o.failAfter > 0 {
		o.failAfter--
	}
	return true
}

func (o *mockParseObserver) notifyLongFlag(name string) bool {
	o.spaceIfNeeded()
	o.b.WriteString("(long ")
	o.b.WriteString(name)
	o.b.WriteString(")")
	return o.injectFault()
}

func (o *mockParseObserver) notifyLongFlagValue(name string, value string) bool {
	o.spaceIfNeeded()
	o.b.WriteString("(long ")
	o.b.WriteString(name)
	o.b.WriteString("=")
	o.b.WriteString(value)
	o.b.WriteString(")")
	return o.injectFault()
}

func (o *mockParseObserver) notifyShortFlag(name rune) bool {
	o.spaceIfNeeded()
	o.b.WriteString("(short ")
	o.b.WriteString(string(name))
	o.b.WriteString(")")
	return o.injectFault()
}

func (o *mockParseObserver) notifyShortFlagValue(name rune, value string) bool {
	o.spaceIfNeeded()
	o.b.WriteString("(short ")
	o.b.WriteString(string(name))
	o.b.WriteString("=")
	o.b.WriteString(value)
	o.b.WriteString(")")
	return o.injectFault()
}

func (o *mockParseObserver) Error(message string) {
	o.spaceIfNeeded()
	o.b.WriteString("(error ")
	o.b.WriteString(message)
	o.b.WriteString(")")
	o.numErrors++
}

func (o *mockParseObserver) NumErrors() int {
	return o.numErrors
}

func (o *mockParseObserver) notifyArg(value string) bool {
	o.spaceIfNeeded()
	o.b.WriteString("(arg ")
	o.b.WriteString(value)
	o.b.WriteString(")")
	return o.injectFault() && !o.banArgs
}

func (o *mockParseObserver) completeLongFlag(prefix string, c CompletionObserver) {
	for _, f := range o.all {
		if f.long != "" && strings.HasPrefix(f.long, prefix) {
			c.FinalCompletion(f.long)
		}
	}
}

func (o *mockParseObserver) completeShortFlag(c CompletionObserver) {
	for _, f := range o.all {
		if f.short != "" {
			if f.hasArg {
				c.FinalCompletion(f.short)
			} else {
				c.PartialCompletion(f.short)
			}
		}
	}
}

func (o *mockParseObserver) completeLongFlagValue(name string, value string, c CompletionObserver) {
}

func (o *mockParseObserver) completeShortFlagValue(name rune, value string, c CompletionObserver) {
}

func (o *mockParseObserver) completeArg(prefix string, c CompletionObserver) {
}

func (o *mockParseObserver) acceptingArgs() bool {
	return !o.banArgs
}

func makeObserver(failAfter int) *mockParseObserver {
	o := &mockParseObserver{
		short:     map[string]*mockFlag{},
		long:      map[string]*mockFlag{},
		failAfter: failAfter,
	}
	o.flag("", "a", false)
	o.flag("", "b", false)
	o.flag("", "c", true)
	o.flag("", "d", true)
	o.flag("foo", "", false)
	o.flag("bar", "", true)
	return o
}

func TestParseArgs(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, true, parse([]string{"abc", "def"}, o))
	assert.Equal(t, "(arg abc) (arg def)", o.b.String())
}

func TestParseArgsFault(t *testing.T) {
	o := makeObserver(0)
	assert.Equal(t, false, parse([]string{"abc", "def"}, o))
	assert.Equal(t, "(arg abc)", o.b.String())
}

func TestParseEscape(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, true, parse([]string{"--", "-a", "--foo"}, o))
	assert.Equal(t, "(arg -a) (arg --foo)", o.b.String())
}

func TestParseShortFlag(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, true, parse([]string{"-a", "-b"}, o))
	assert.Equal(t, "(short a) (short b)", o.b.String())
}

func TestParseShortFault(t *testing.T) {
	o := makeObserver(0)
	assert.Equal(t, false, parse([]string{"-a", "-b"}, o))
	assert.Equal(t, "(short a)", o.b.String())
}

func TestParseShortFlagCombine(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, true, parse([]string{"-ab"}, o))
	assert.Equal(t, "(short a) (short b)", o.b.String())
}

func TestParseShortFlagCombineFault(t *testing.T) {
	o := makeObserver(1)
	assert.Equal(t, false, parse([]string{"-aba"}, o))
	assert.Equal(t, "(short a) (short b)", o.b.String())
}

func TestParseShortValue(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, true, parse([]string{"-c", "12", "-d", "34"}, o))
	assert.Equal(t, "(short c=12) (short d=34)", o.b.String())
}

func TestParseShortValueFault(t *testing.T) {
	o := makeObserver(0)
	assert.Equal(t, false, parse([]string{"-c", "12", "-d", "34"}, o))
	assert.Equal(t, "(short c=12)", o.b.String())
}

func TestParseShortValueNoArg(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, false, parse([]string{"-c"}, o))
	assert.Equal(t, "(error -c requires an argument)", o.b.String())
}

func TestParseShortUnrecognized(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, false, parse([]string{"-e"}, o))
	assert.Equal(t, "(error unrecognized flag -e)", o.b.String())
}

func TestParseShortValueComine(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, true, parse([]string{"-ac", "12"}, o))
	assert.Equal(t, "(short a) (short c=12)", o.b.String())
}

func TestParseCombineUnrecognized(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, false, parse([]string{"-ae"}, o))
	assert.Equal(t, "(short a) (error unrecognized flag -e)", o.b.String())
}

func TestParseLong(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, true, parse([]string{"--foo"}, o))
	assert.Equal(t, "(long foo)", o.b.String())
}

func TestParseLongFault(t *testing.T) {
	o := makeObserver(0)
	assert.Equal(t, false, parse([]string{"--foo"}, o))
	assert.Equal(t, "(long foo)", o.b.String())
}

func TestParseLongValue(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, true, parse([]string{"--bar", "abc"}, o))
	assert.Equal(t, "(long bar=abc)", o.b.String())
}

func TestParseLongValueFault(t *testing.T) {
	o := makeObserver(0)
	assert.Equal(t, false, parse([]string{"--bar", "abc"}, o))
	assert.Equal(t, "(long bar=abc)", o.b.String())
}

func TestParseLongValueCombine(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, true, parse([]string{"--bar=abc"}, o))
	assert.Equal(t, "(long bar=abc)", o.b.String())
}

func TestParseLongValueCombineFault(t *testing.T) {
	o := makeObserver(0)
	assert.Equal(t, false, parse([]string{"--bar=abc"}, o))
	assert.Equal(t, "(long bar=abc)", o.b.String())
}

func TestParseLongValueNoArg(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, false, parse([]string{"--bar"}, o))
	assert.Equal(t, "(error --bar requires an argument)", o.b.String())
}

func TestParseLongUnrecognized(t *testing.T) {
	o := makeObserver(-1)
	assert.Equal(t, false, parse([]string{"--baz"}, o))
	assert.Equal(t, "(error unrecognized flag --baz)", o.b.String())
}

func TestCompleteLong(t *testing.T) {
	o := makeObserver(-1)
	options, _ := complete([]string{"--"}, o)
	assert.Equal(t, []string{"--", "--foo", "--bar"}, options)
}

func TestCompleteLongNoArgs(t *testing.T) {
	o := makeObserver(-1)
	o.banArgs = true
	options, _ := complete([]string{"--"}, o)
	assert.Equal(t, []string{"--foo", "--bar"}, options)
}

func TestCompleteLongPartial(t *testing.T) {
	o := makeObserver(-1)
	options, _ := complete([]string{"--b"}, o)
	assert.Equal(t, []string{"--bar"}, options)
}

func TestCompleteLongExact(t *testing.T) {
	o := makeObserver(-1)
	options, _ := complete([]string{"--bar"}, o)
	assert.Equal(t, []string{"--bar"}, options)
}

func TestCompleteShort(t *testing.T) {
	o := makeObserver(-1)
	options, _ := complete([]string{"-"}, o)
	assert.Equal(t, []string{"-a", "-b", "-c", "-d", "--", "--foo", "--bar"}, options)
}

func TestCompleteShortNoArgs(t *testing.T) {
	o := makeObserver(-1)
	o.banArgs = true
	options, _ := complete([]string{"-"}, o)
	assert.Equal(t, []string{"-a", "-b", "-c", "-d", "--foo", "--bar"}, options)
}

func TestCompleteShortNoValue(t *testing.T) {
	o := makeObserver(-1)
	options, _ := complete([]string{"-a"}, o)
	assert.Equal(t, []string{"-a", "-aa", "-ab", "-ac", "-ad"}, options)
}

func TestCompleteShortYesValue(t *testing.T) {
	o := makeObserver(-1)
	options, _ := complete([]string{"-c"}, o)
	assert.Equal(t, []string{"-c"}, options)
}

func TestCompleteEmptyNoArgs(t *testing.T) {
	o := makeObserver(-1)
	o.banArgs = true
	options, _ := complete([]string{""}, o)
	assert.Equal(t, []string{"-a", "-b", "-c", "-d", "--foo", "--bar"}, options)
}
