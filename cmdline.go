package cmdline

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Logger interface {
	Error(message string)
	NumErrors() int
}

func SetTrue(value *bool) func() {
	return func() {
		*value = true
	}
}

func SetFalse(value *bool) func() {
	return func() {
		*value = false
	}
}

type Flag struct {
	Long     string
	Short    rune
	Value    ValueHandler
	Call     func()
	Default  string
	Min      int
	Max      int
	useCount int
}

func (f *Flag) Name() string {
	if f.Long != "" {
		if f.Short != 0 {
			return "-" + string(f.Short) + "/--" + f.Long
		} else {
			return "--" + f.Long
		}
	} else if f.Short != 0 {
		return "-" + string(f.Short)
	} else {
		panic("Flag has no name?")
	}
}

func (f *Flag) Required() *Flag {
	if f.Min < 1 {
		f.Min = 1
	}
	return f
}

func (f *Flag) CanAcceptMore() bool {
	return f.useCount < f.Max
}

type Argument struct {
	Name  string
	Value ValueHandler
}

func (a *Argument) ArgumentValue(handler ValueHandler) *Argument {
	a.Value = handler
	return a
}

type App struct {
	name              string
	allFlags          []*Flag
	longToFlag        map[string]*Flag
	shortToFlag       map[rune]*Flag
	requiredArguments []*Argument
	excessArguments   *Argument
	currentArgument   int
	completing        bool
	numErrors         int
}

func (app *App) indexFlag(flag *Flag) {
	app.allFlags = append(app.allFlags, flag)
	if flag.Long != "" {
		_, ok := app.longToFlag[flag.Long]
		if ok {
			panic("Tried to redefine --" + flag.Long)
		}
		app.longToFlag[flag.Long] = flag
	}
	if flag.Short != 0 {
		_, ok := app.shortToFlag[flag.Short]
		if ok {
			panic("Tried to redefine -" + string(flag.Short))
		}
		app.shortToFlag[flag.Short] = flag
	}
}

func (app *App) Flags(flags []*Flag) {
	for _, flag := range flags {
		if flag.Long == "" && flag.Short == 0 {
			panic("Flag has no name.")
		}
		app.indexFlag(flag)
		if flag.Value == nil && flag.Call == nil {
			panic(flag.Name() + " has no effect.")
		}
		if flag.Value == nil && flag.Default != "" {
			panic(flag.Name() + " default value but not value handler.")
		}
	}
}

func (app *App) RequiredArgs(args []*Argument) {
	for _, a := range args {
		app.requiredArguments = append(app.requiredArguments, a)
	}
}

func (app *App) ExcessArguments(arg *Argument) {
	app.excessArguments = arg
}

func (app *App) WriteHelp(out io.Writer) {
	io.WriteString(out, "usage: ")
	io.WriteString(out, app.name)
	if len(app.allFlags) > 0 {
		io.WriteString(out, " [<flags>]")
	}
	for _, a := range app.requiredArguments {
		io.WriteString(out, " <")
		io.WriteString(out, a.Name)
		io.WriteString(out, ">")
	}

	if app.excessArguments != nil {
		a := app.excessArguments
		io.WriteString(out, " [<")
		io.WriteString(out, a.Name)
		io.WriteString(out, ">...]")
	}
	out.Write([]byte("\n"))

	if len(app.allFlags) > 0 {
		io.WriteString(out, "\n")
		io.WriteString(out, "Flags:\n")
		for _, f := range app.allFlags {
			io.WriteString(out, "    ")
			io.WriteString(out, f.Name())
			if f.Value != nil {
				io.WriteString(out, "   ")
				io.WriteString(out, f.Value.TypeName())
			}
			if f.Default != "" {
				io.WriteString(out, "   default=")
				io.WriteString(out, f.Default)
			}
			if f.Min > 0 {
				io.WriteString(out, "   required")
			}
			io.WriteString(out, "\n")
		}
	}

	if len(app.requiredArguments) > 0 {
		displayArg := func(name string, a *Argument) {
			io.WriteString(out, "    ")
			io.WriteString(out, name)
			if a.Value != nil {
				io.WriteString(out, "   ")
				io.WriteString(out, a.Value.TypeName())
			}
			io.WriteString(out, "\n")

		}

		io.WriteString(out, "\n")
		io.WriteString(out, "Args:\n")
		for _, a := range app.requiredArguments {
			displayArg(a.Name, a)
		}
		if app.excessArguments != nil {
			a := app.excessArguments
			displayArg("<"+a.Name+">...", a)
		}
	}
}

func (app *App) longFlagInfo(name string) (bool, bool) {
	flag, ok := app.longToFlag[name]
	if ok {
		return true, flag.Value != nil
	} else {
		return false, false
	}
}

func (app *App) shortFlagInfo(name rune) (bool, bool) {
	flag, ok := app.shortToFlag[name]
	if ok {
		return true, flag.Value != nil
	} else {
		return false, false
	}
}

func (app *App) notifyLongFlag(name string) bool {
	f := app.longToFlag[name]
	f.useCount++
	f.Call()
	return true
}

func (app *App) notifyLongFlagValue(name string, value string) bool {
	f := app.longToFlag[name]
	f.useCount++
	return f.Value.Notify(value, app)
}

func (app *App) notifyShortFlag(name rune) bool {
	f := app.shortToFlag[name]
	f.useCount++
	f.Call()
	return true
}

func (app *App) notifyShortFlagValue(name rune, value string) bool {
	f := app.shortToFlag[name]
	f.useCount++
	return f.Value.Notify(value, app)
}

func (app *App) notifyArg(value string) bool {
	if app.currentArgument < len(app.requiredArguments) {
		a := app.requiredArguments[app.currentArgument]
		app.currentArgument++
		return a.Value.Notify(value, app)
	} else if app.excessArguments != nil {
		return app.excessArguments.Value.Notify(value, app)
	}
	app.Error("Extra argument: " + value)
	return true
}

func (app *App) Error(message string) {
	if !app.completing {
		fmt.Println("ERROR", message)
	}
	app.numErrors++
}

func (app *App) NumErrors() int {
	return app.numErrors
}

func (app *App) completeLongFlag(prefix string, c CompletionObserver) {
	for _, f := range app.allFlags {
		if !f.CanAcceptMore() {
			continue
		}
		if f.Long != "" && strings.HasPrefix(f.Long, prefix) {
			c.FinalCompletion(f.Long)
		}
	}
}

func (app *App) completeShortFlag(c CompletionObserver) {
	for _, f := range app.allFlags {
		if !f.CanAcceptMore() {
			continue
		}
		if f.Short != 0 {
			if f.Value != nil {
				c.FinalCompletion(string(f.Short))
			} else {
				c.PartialCompletion(string(f.Short))
			}
		}
	}
}

func (app *App) completeLongFlagValue(name string, value string, c CompletionObserver) {
	app.longToFlag[name].Value.Complete(value, c)
}

func (app *App) completeShortFlagValue(name rune, value string, c CompletionObserver) {
	app.shortToFlag[name].Value.Complete(value, c)
}

func (app *App) completeArg(prefix string, c CompletionObserver) {
	var a *Argument
	if app.currentArgument < len(app.requiredArguments) {
		a = app.requiredArguments[app.currentArgument]
	} else if app.excessArguments != nil {
		a = app.excessArguments
	}
	a.Value.Complete(prefix, c)
}

func (app *App) acceptingArgs() bool {
	return app.currentArgument < len(app.requiredArguments) || app.excessArguments != nil
}

func (app *App) postParse() bool {
	for _, f := range app.allFlags {
		if f.Default != "" && f.useCount == 0 {
			f.Value.Notify(f.Default, app)
		}
		if f.Min > f.useCount {
			app.Error(f.Name() + " is required")
		}
	}
	for i := app.currentArgument; i < len(app.requiredArguments); i++ {
		app.Error(fmt.Sprintf("argument %#v is required", app.requiredArguments[i].Name))
	}
	return app.NumErrors() == 0
}

const scriptTemplate = `# Usage: eval "$(%s --bash-completion-script)"
_%s_bash_autocomplete() {
    local cur args opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    COMP_WORDS+=("")
    args=("${COMP_WORDS[0]}" "--generate-bash-completion" "${COMP_WORDBREAKS}" "${COMP_WORDS[@]:1:$COMP_CWORD}")
    opts=$("${args[@]}")
		local IFS=$'\n'
    COMPREPLY=($(compgen -W "${opts}"))
    return 0
}
complete -o nospace -F _%s_bash_autocomplete %s
`

func completionClipPoint(prefix string, compWordbreaks string) int {
	breaks := []rune(compWordbreaks)
	chars := []rune(prefix)
	for i := len(chars) - 1; i >= 0; i-- {
		for _, r := range breaks {
			if chars[i] == r {
				return len(string(chars[:i+1]))
			}
		}
	}
	return 0
}

func (app *App) Run(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "--generate-bash-completion":
			app.completing = true
			options, partial := complete(args[2:], app)
			clipPoint := completionClipPoint(args[len(args)-1], args[1])
			if len(options) == 1 && !partial {
				fmt.Println(options[0][clipPoint:] + " ")
			} else {
				for _, o := range options {
					fmt.Println(o[clipPoint:])
				}
			}
			os.Exit(0)
		case "--bash-completion-script":
			fmt.Printf(scriptTemplate, app.name, app.name, app.name, app.name)
			os.Exit(0)
		}
	}
	ok := parse(args, app)
	if ok {
		ok = app.postParse()
	}
	if !ok {
		os.Stdout.WriteString("\n")
		app.WriteHelp(os.Stdout)
		os.Exit(1)
	}
}

func MakeApp(name string) *App {
	a := &App{
		name:        name,
		allFlags:    []*Flag{},
		longToFlag:  map[string]*Flag{},
		shortToFlag: map[rune]*Flag{},
	}
	return a
}
