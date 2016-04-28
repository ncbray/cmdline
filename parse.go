package cmdline

type parseObserver interface {
	Logger

	longFlagInfo(name string) (bool, bool)
	shortFlagInfo(name rune) (bool, bool)

	notifyLongFlag(name string) bool
	notifyLongFlagValue(name string, value string) bool
	notifyShortFlag(name rune) bool
	notifyShortFlagValue(name rune, value string) bool
	notifyArg(value string) bool

	completeLongFlag(prefix string, c CompletionObserver)
	completeLongFlagValue(name string, value string, c CompletionObserver)

	completeShortFlag(c CompletionObserver)
	completeShortFlagValue(name rune, value string, c CompletionObserver)

	completeArg(prefix string, c CompletionObserver)
	acceptingArgs() bool
}

type CompletionObserver interface {
	PartialCompletion(completion string)
	FinalCompletion(completion string)
}

type parser struct {
	args       []string
	current    int
	parseOK    bool
	completing bool

	prependCompletion string
	completions       []string
	isPartial         bool
}

func (p *parser) hasNext() bool {
	return p.current < len(p.args)
}

func (p *parser) getNext() string {
	temp := p.args[p.current]
	p.current++
	return temp
}

func (p *parser) shouldComplete() bool {
	return p.completing && !p.hasNext()
}

func (p *parser) status(ok bool) {
	if !ok {
		p.parseOK = false
	}
}

func (p *parser) FinalCompletion(completion string) {
	p.completions = append(p.completions, p.prependCompletion+completion)
}

func (p *parser) PartialCompletion(completion string) {
	p.completions = append(p.completions, p.prependCompletion+completion)
	p.isPartial = true
}

func handleLongFlagValue(p *parser, name string, value string, observer parseObserver) {
	if p.shouldComplete() {
		observer.completeLongFlagValue(name, value, p)
	} else {
		p.status(observer.notifyLongFlagValue(name, value))
	}
}

func parseLongFlag(p *parser, arg []rune, observer parseObserver) {
	c := 0
	equals := false
	for c < len(arg) {
		if arg[c] == '=' {
			equals = true
			break
		}
		c++
	}
	name := string(arg[:c])
	exists, takesValue := observer.longFlagInfo(name)

	if equals {
		if !exists {
			observer.Error("unrecognized flag --" + name)
			p.status(false)
		} else if takesValue {
			value := string(arg[c+1:])
			p.prependCompletion = "--" + string(arg[:c+1])
			//p.prependCompletion = ""
			handleLongFlagValue(p, name, value, observer)
		} else {
			observer.Error("--" + name + " does not take an argument")
			p.status(false)
		}
	} else {
		if p.shouldComplete() {
			completeLongFlag(p, name, observer)
		} else if !exists {
			observer.Error("unrecognized flag --" + name)
			p.status(false)
		} else if takesValue {
			if p.hasNext() {
				value := p.getNext()
				p.prependCompletion = ""
				handleLongFlagValue(p, name, value, observer)
			} else {
				observer.Error("--" + name + " requires an argument")
				p.status(false)
			}
		} else {
			p.status(observer.notifyLongFlag(name))
		}
	}
}

func handleShortFlagValue(p *parser, name rune, value string, observer parseObserver) {
	if p.shouldComplete() {
		observer.completeShortFlagValue(name, value, p)
	} else {
		p.status(observer.notifyShortFlagValue(name, value))
	}
}

func parseShortFlag(p *parser, arg []rune, observer parseObserver) {
	for c := 0; c < len(arg) && p.parseOK; c++ {
		name := arg[c]
		exists, takesValue := observer.shortFlagInfo(name)
		if !exists {
			observer.Error("unrecognized flag -" + string(name))
			p.status(false)
		} else if takesValue {
			c++
			if c < len(arg) {
				value := string(arg[c:])
				handleShortFlagValue(p, name, value, observer)
			} else if p.hasNext() {
				value := p.getNext()
				handleShortFlagValue(p, name, value, observer)
			} else {
				if p.shouldComplete() {
					p.FinalCompletion("-" + string(arg))
				} else {
					observer.Error("-" + string(name) + " requires an argument")
					p.status(false)
				}
			}
			return
		} else {
			p.status(observer.notifyShortFlag(name))
		}
	}

	// Last flag does not require an argument
	if p.parseOK && p.shouldComplete() {
		completeShortFlag(p, "-"+string(arg), observer)
	}
}

func completeLongFlag(p *parser, prefix string, observer parseObserver) {
	p.prependCompletion = "--"
	if prefix == "" && observer.acceptingArgs() {
		p.FinalCompletion("")
	}
	observer.completeLongFlag(prefix, p)
}

func completeShortFlag(p *parser, current string, observer parseObserver) {
	p.prependCompletion = current
	if len(current) > 1 {
		p.FinalCompletion("")
	}
	observer.completeShortFlag(p)
}

func completeAnyFlag(p *parser, observer parseObserver) {
	completeShortFlag(p, "-", observer)
	completeLongFlag(p, "", observer)
}

func parseMain(p *parser, observer parseObserver) {
	for p.hasNext() && p.parseOK {
		arg := []rune(p.getNext())
		if len(arg) >= 1 && arg[0] == '-' {
			if len(arg) >= 2 {
				if arg[1] == '-' {
					if len(arg) >= 3 {
						parseLongFlag(p, arg[2:], observer)
					} else {
						if p.shouldComplete() {
							completeLongFlag(p, "", observer)
						} else {
							// Do not treat arguments after "--" as flags.
							for p.hasNext() && p.parseOK {
								p.status(observer.notifyArg(p.getNext()))
							}
						}
					}
				} else {
					parseShortFlag(p, arg[1:], observer)
				}
			} else {
				if p.shouldComplete() {
					completeAnyFlag(p, observer)
				} else {
					observer.Error("stray dash")
					p.status(false)
				}
			}
		} else {
			if p.shouldComplete() {
				if len(arg) == 0 && !observer.acceptingArgs() {
					completeAnyFlag(p, observer)
				} else {
					observer.completeArg(string(arg), p)
				}
			} else {
				// Not a flag, must be an argument.
				p.status(observer.notifyArg(string(arg)))
			}
		}
	}
}

func parse(args []string, observer parseObserver) bool {
	p := &parser{args: args, current: 0, parseOK: true}
	parseMain(p, observer)
	return p.parseOK && observer.NumErrors() == 0
}

func complete(args []string, observer parseObserver) ([]string, bool) {
	p := &parser{args: args, current: 0, parseOK: true, completing: true}
	parseMain(p, observer)
	return p.completions, p.isPartial
}
