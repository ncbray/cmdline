package cmdline

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type StringHandlerFactory interface {
	Set(ptr *string) ValueHandler
	Call(func(value string)) ValueHandler
}

type StringParser interface {
	Parse(text string) (string, error)
	Complete(text string, observer CompletionObserver)
	TypeName() string
}

type SimpleStringParser struct {
	StringHandlerFactory
}

func (p *SimpleStringParser) Parse(text string) (string, error) {
	return text, nil
}

func (p *SimpleStringParser) Complete(text string, observer CompletionObserver) {
}

func (p *SimpleStringParser) TypeName() string {
	return "string"
}

func (p *SimpleStringParser) Set(ptr *string) ValueHandler {
	return &StringHandler{Parser: p, Ptr: ptr}
}

func (p *SimpleStringParser) Call(callback func(value string)) ValueHandler {
	return &StringHandler{Parser: p, Callback: callback}
}

var String StringHandlerFactory = &SimpleStringParser{}

type StringHandler struct {
	Parser         StringParser
	Callback       func(value string)
	Ptr            *string
	AffectsParsing bool
}

func (h *StringHandler) Notify(text string, log Logger) bool {
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

func (h *StringHandler) Complete(text string, observer CompletionObserver) {
	h.Parser.Complete(text, observer)
}

func (h *StringHandler) TypeName() string {
	return h.Parser.TypeName()
}

type Enum struct {
	Possible []string
}

func (p *Enum) Parse(text string) (string, error) {
	for _, possible := range p.Possible {
		if text == possible {
			return text, nil
		}
	}
	return "", &parseError{message: fmt.Sprintf("%#v is not in %s", text, p.TypeName())}
}

func (p *Enum) Complete(text string, observer CompletionObserver) {
	for _, possible := range p.Possible {
		if strings.HasPrefix(possible, text) {
			observer.FinalCompletion(possible)
		}
	}
}

func (p *Enum) TypeName() string {
	return fmt.Sprintf("{%s}", strings.Join(p.Possible, ","))
}

func (p *Enum) Set(ptr *string) ValueHandler {
	return &StringHandler{Parser: p, Ptr: ptr}
}

func (p *Enum) Call(callback func(value string)) ValueHandler {
	return &StringHandler{Parser: p, Callback: callback}
}

type FilePath struct {
	Root       string
	MustExist  bool
	FileFilter func(os.FileInfo) bool
}

func (p *FilePath) effectivePath(file string) string {
	if p.Root != "" {
		return filepath.Join(p.Root, file)
	} else if file != "" {
		return file
	} else {
		return "."
	}
}

func (p *FilePath) Parse(text string) (string, error) {
	fullpath := p.effectivePath(text)
	_, err := os.Stat(fullpath)
	if err != nil {
		if p.MustExist && os.IsNotExist(err) {
			return "", &parseError{message: fullpath + ": no such file or directory"}
		}
	}
	return text, nil
}

func (p *FilePath) Complete(text string, observer CompletionObserver) {
	dir, prefix := filepath.Split(text)
	files, err := ioutil.ReadDir(p.effectivePath(dir))
	// If this path isn't rooted in a real directory, don't offer any completions.
	if err != nil {
		return
	}
	for _, file := range files {
		name := file.Name()
		// Is it a valid completion?
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		// Does the application accept it as a completion?
		if p.FileFilter != nil && !p.FileFilter(file) {
			continue
		}
		// Generate the completion.
		full := filepath.Join(dir, name)
		if file.IsDir() {
			observer.PartialCompletion(full + "/")
		} else {
			observer.FinalCompletion(full)
		}
	}
}

func (p *FilePath) TypeName() string {
	name := ""
	if p.MustExist {
		name = "existing file"
	} else {
		name = "file path"
	}
	if p.Root != "" {
		name += " in " + p.Root
	}
	return name
}

func (p *FilePath) Set(ptr *string) ValueHandler {
	return &StringHandler{Parser: p, Ptr: ptr}
}

func (p *FilePath) Call(callback func(value string)) ValueHandler {
	return &StringHandler{Parser: p, Callback: callback}
}
