package main

import (
	"fmt"
	"github.com/ncbray/cmdline"
	"os"
)

func main() {
	var foo bool
	var bar int32
	var verbosity int32
	var jobs int32
	var arch string

	archEnum := &cmdline.Enum{Possible: []string{"arm", "arm64", "ia32", "x64"}}

	app := cmdline.MakeApp("cmdline_playground")
	app.Flags([]*cmdline.Flag{
		{
			Long:  "foo",
			Short: 'f',
			Call:  cmdline.SetTrue(&foo),
		},
		{
			Long:  "bar",
			Short: 'b',
			Value: cmdline.Int32.Set(&bar),
			Min:   1,
			Max:   1,
		},
		{
			Long:    "verbosity",
			Short:   'v',
			Value:   cmdline.Int32.Set(&verbosity),
			Default: "0",
		},
		{
			Long:    "jobs",
			Short:   'j',
			Value:   cmdline.Int32.Set(&jobs),
			Default: "32",
		},
		{
			Long:    "arch",
			Value:   archEnum.Set(&arch),
			Default: "arm64",
		},
	})

	app.Run(os.Args[1:])

	fmt.Println("foo", foo)
	fmt.Println("bar", bar)
	fmt.Println("verbosity", verbosity)
	fmt.Println("jobs", jobs)
	fmt.Println("arch", arch)
}
