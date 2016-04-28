package cmdline

import (
	"fmt"
	"strconv"
)

type Int32HandlerFactory interface {
	Set(ptr *int32) ValueHandler
	Call(func(value int32)) ValueHandler
}

type Int32Parser interface {
	Parse(text string) (int32, error)
	Complete(text string, observer CompletionObserver)
	TypeName() string
}

type SimpleInt32Parser struct {
	Int32HandlerFactory
}

func (p *SimpleInt32Parser) Parse(text string) (int32, error) {
	value, err := strconv.ParseInt(text, 0, 32)
	if err != nil {
		return 0, &parseError{message: fmt.Sprintf("%#v cannot be converted into an int32", text)}
	}
	return int32(value), nil
}

func (p *SimpleInt32Parser) Complete(text string, observer CompletionObserver) {
}

func (p *SimpleInt32Parser) TypeName() string {
	return "int32"
}

func (p *SimpleInt32Parser) Set(ptr *int32) ValueHandler {
	return &Int32Handler{Parser: p, Ptr: ptr}
}

func (p *SimpleInt32Parser) Call(callback func(value int32)) ValueHandler {
	return &Int32Handler{Parser: p, Callback: callback}
}

var Int32 Int32HandlerFactory = &SimpleInt32Parser{}

type Int32Handler struct {
	Parser         Int32Parser
	Callback       func(value int32)
	Ptr            *int32
	AffectsParsing bool
}

func (h *Int32Handler) Notify(text string, log Logger) bool {
	value, err := h.Parser.Parse(text)
	if err != nil {
		log.Error(err.Error())
		return !h.AffectsParsing
	} else if h.Callback != nil {
		h.Callback(value)
		return true
	} else if h.Ptr != nil {
		*h.Ptr = value
		return true
	} else {
		log.Error("missing consumer")
		return !h.AffectsParsing
	}
}

func (h *Int32Handler) Complete(text string, observer CompletionObserver) {
	h.Parser.Complete(text, observer)
}

func (h *Int32Handler) TypeName() string {
	return h.Parser.TypeName()
}
