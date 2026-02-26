package main

import (
	"fmt"
	"os"

	"github.com/sampbb1996-dot/tsr/internal/kernel"
	"github.com/sampbb1996-dot/tsr/internal/tsr"
)

func main() {
	args := os.Args[1:]

	trace := false
	tracePolicy := false
	var filteredArgs []string
	for _, a := range args {
		switch a {
		case "--trace":
			trace = true
		case "--trace-policy":
			tracePolicy = true
		default:
			filteredArgs = append(filteredArgs, a)
		}
	}
	args = filteredArgs

	if len(args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := args[0]
	file := args[1]

	switch cmd {
	case "run":
		code, stdout, stderr := tsr.RunFile(file, trace, tracePolicy)
		if stdout != "" {
			fmt.Print(stdout)
		}
		if stderr != "" {
			fmt.Fprint(os.Stderr, stderr)
		}
		os.Exit(code)

	case "compile":
		tsrSource, warnings, policyTrace, err := kernel.CompileFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "compile error: %s\n", err)
			os.Exit(1)
		}
		if tracePolicy && policyTrace != "" {
			fmt.Fprintln(os.Stderr, policyTrace)
		}
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		}
		fmt.Print(tsrSource)

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "TSR - a governed programming language")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  tsr run [--trace] [--trace-policy] <file.tsr>    Run a TSR program")
	fmt.Fprintln(os.Stderr, "  tsr compile [--trace-policy] <file.krn>           Compile KERNEL to TSR (emits to stdout)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  tsr run examples/00_hello.tsr")
	fmt.Fprintln(os.Stderr, "  tsr compile examples/20_kernel_demo.krn > /tmp/demo.tsr")
	fmt.Fprintln(os.Stderr, "  tsr run /tmp/demo.tsr")
	fmt.Fprintln(os.Stderr, "  tsr compile --trace-policy examples/30_context_prod.krn > /tmp/prod.tsr")
}
