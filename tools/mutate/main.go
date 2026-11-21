// MIT License
//
// Copyright (C) 2026 John Kleijn
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//

// Command mutate is a mutation tester for concurrent Go code. It complements operator-level tools such as
// gremlins with mutations of the concurrency mechanism itself: a missing close, a select without its
// cancellation or default case, an unlocked critical section, a missing wait or cancel, a goroutine that runs
// synchronously, and a channel with a different buffer.
//
// Every mutant is compiled and tested with `go test -race` through an overlay, so the source files are never
// modified and mutants run in parallel. A mutant is killed when the tests fail or time out, lives when they
// pass, and is not viable when it does not compile.
//
// Usage:
//
//	mutate [-timeout 30s] [-workers N] [-v] dir...
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

type edit struct {
	text       string
	start, end int
}

type mutant struct {
	file   string
	op     string
	status string
	edits  []edit
	pos    token.Position
}

const (
	killed    = "KILLED"
	timedOut  = "TIMED OUT"
	lived     = "LIVED"
	notViable = "NOT VIABLE"
)

func main() {
	timeout := flag.Duration("timeout", 30*time.Second, "test timeout per mutant")
	workers := flag.Int("workers", max(1, runtime.GOMAXPROCS(0)/2), "mutants tested at the same time")
	verbose := flag.Bool("v", false, "print every mutant, not only the ones that lived")
	flag.Parse()

	if flag.NArg() == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: mutate [-timeout d] [-workers n] [-v] dir...")
		os.Exit(2)
	}

	var all []*mutant

	for _, dir := range flag.Args() {
		ms, err := findMutants(dir)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "mutate:", err)
			os.Exit(1)
		}

		all = append(all, ms...)
	}

	tmp, err := os.MkdirTemp("", "mutate-")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "mutate:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp) //nolint:errcheck // best effort

	jobs := make(chan int)

	var wg sync.WaitGroup

	for range *workers {
		wg.Go(func() {
			for i := range jobs {
				all[i].status = run(context.Background(), tmp, i, all[i], *timeout)
			}
		})
	}

	for i := range all {
		jobs <- i
	}

	close(jobs)
	wg.Wait()

	report(all, *verbose)
}

// findMutants lists the mutants of the non-test files in dir.
func findMutants(dir string) ([]*mutant, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	files, err := filepath.Glob(filepath.Join(abs, "*.go"))
	if err != nil {
		return nil, err
	}

	var out []*mutant

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		ms, findErr := mutantsOf(file)
		if findErr != nil {
			return nil, findErr
		}

		out = append(out, ms...)
	}

	return out, nil
}

func mutantsOf(file string) ([]*mutant, error) {
	src, err := os.ReadFile(file) //nolint:gosec // paths come from globbing the given directories
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, file, src, 0)
	if err != nil {
		return nil, err
	}

	off := func(p token.Pos) int { return fset.Position(p).Offset }
	text := func(n ast.Node) string { return string(src[off(n.Pos()):off(n.End())]) }

	var out []*mutant

	add := func(at ast.Node, op string, edits ...edit) {
		out = append(out, &mutant{file: file, pos: fset.Position(at.Pos()), op: op, edits: edits})
	}

	remove := func(n ast.Node) edit { return edit{start: off(n.Pos()), end: off(n.End())} }

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BlockStmt:
			blockMutants(x.List, text, remove, add)
		case *ast.CaseClause:
			blockMutants(x.Body, text, remove, add)
		case *ast.CommClause:
			blockMutants(x.Body, text, remove, add)
		case *ast.SelectStmt:
			selectMutants(x, remove, add)
		case *ast.GoStmt:
			add(x, "go-to-sync", edit{start: off(x.Go), end: off(x.Call.Pos())})
		case *ast.IfStmt:
			if _, isBinary := x.Cond.(*ast.BinaryExpr); !isBinary {
				add(x.Cond, "negate-condition",
					edit{start: off(x.Cond.Pos()), end: off(x.Cond.Pos()), text: "!("},
					edit{start: off(x.Cond.End()), end: off(x.Cond.End()), text: ")"})
			}
		case *ast.CallExpr:
			makeMutants(x, off, add)
		}

		return true
	})

	return out, nil
}

// blockMutants removes statements that carry the synchronization of a block.
func blockMutants(
	list []ast.Stmt, text func(ast.Node) string, remove func(ast.Node) edit, add func(ast.Node, string, ...edit),
) {
	for i, stmt := range list {
		call, deferred := stmtCall(stmt)
		if call == nil {
			continue
		}

		name, recv := callName(call, text)

		switch {
		case name == "close" && recv == "":
			add(stmt, "remove-close", remove(stmt))
		case name == "Wait" && len(call.Args) == 0 && !deferred:
			add(stmt, "remove-wait", remove(stmt))
		case recv == "" && strings.Contains(strings.ToLower(name), "cancel"):
			add(stmt, "remove-cancel", remove(stmt))
		case (name == "Lock" || name == "RLock") && recv != "" && !deferred:
			unlock := "Unlock"
			if name == "RLock" {
				unlock = "RUnlock"
			}

			for _, later := range list[i+1:] {
				c, _ := stmtCall(later)
				if c == nil {
					continue
				}

				if n, r := callName(c, text); n == unlock && r == recv {
					add(stmt, "remove-critical-section", remove(stmt), remove(later))

					break
				}
			}
		}
	}
}

// selectMutants removes the cancellation case or the default case of a select.
func selectMutants(s *ast.SelectStmt, remove func(ast.Node) edit, add func(ast.Node, string, ...edit)) {
	if len(s.Body.List) < 2 {
		return
	}

	for _, c := range s.Body.List {
		clause, ok := c.(*ast.CommClause)
		if !ok {
			continue
		}

		switch {
		case clause.Comm == nil:
			add(clause, "remove-default", remove(clause))
		case receivesFromDone(clause.Comm):
			add(clause, "remove-done-case", remove(clause))
		}
	}
}

// makeMutants changes the buffer of a channel: buffered to unbuffered, and unbuffered to a buffer of one.
func makeMutants(call *ast.CallExpr, off func(token.Pos) int, add func(ast.Node, string, ...edit)) {
	if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "make" || len(call.Args) == 0 {
		return
	}

	if _, ok := call.Args[0].(*ast.ChanType); !ok {
		return
	}

	switch len(call.Args) {
	case 1:
		add(call, "buffer-channel", edit{start: off(call.Args[0].End()), end: off(call.Args[0].End()), text: ", 1"})
	case 2:
		add(call, "unbuffer-channel", edit{start: off(call.Args[0].End()), end: off(call.Args[1].End())})
	}
}

func stmtCall(stmt ast.Stmt) (call *ast.CallExpr, deferred bool) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		c, _ := s.X.(*ast.CallExpr)

		return c, false
	case *ast.DeferStmt:
		return s.Call, true
	}

	return nil, false
}

// callName returns the name of the called function and the text of its receiver, if it is a method call.
func callName(call *ast.CallExpr, text func(ast.Node) string) (name, recv string) {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name, ""
	case *ast.SelectorExpr:
		return fn.Sel.Name, text(fn.X)
	}

	return "", ""
}

func receivesFromDone(comm ast.Stmt) bool {
	var x ast.Expr

	switch s := comm.(type) {
	case *ast.ExprStmt:
		x = s.X
	case *ast.AssignStmt:
		if len(s.Rhs) == 1 {
			x = s.Rhs[0]
		}
	}

	u, ok := x.(*ast.UnaryExpr)
	if !ok || u.Op != token.ARROW {
		return false
	}

	call, ok := u.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)

	return ok && sel.Sel.Name == "Done"
}

// apply returns the source with the edits of the mutant applied.
func apply(src []byte, edits []edit) []byte {
	sorted := slices.Clone(edits)
	slices.SortFunc(sorted, func(a, b edit) int { return b.start - a.start })

	out := slices.Clone(src)
	for _, e := range sorted {
		out = slices.Concat(out[:e.start], []byte(e.text), out[e.end:])
	}

	return out
}

// run tests one mutant and returns its status.
func run(ctx context.Context, tmp string, id int, m *mutant, timeout time.Duration) string {
	src, err := os.ReadFile(m.file)
	if err != nil {
		return notViable
	}

	dir := filepath.Join(tmp, fmt.Sprint(id))
	if err = os.MkdirAll(dir, 0o750); err != nil {
		return notViable
	}

	mutated := filepath.Join(dir, filepath.Base(m.file))
	//nolint:gosec // mutated is in a temporary directory of our own
	if err = os.WriteFile(mutated, apply(src, m.edits), 0o600); err != nil {
		return notViable
	}

	overlay, err := json.Marshal(map[string]map[string]string{"Replace": {m.file: mutated}})
	if err != nil {
		return notViable
	}

	overlayFile := filepath.Join(dir, "overlay.json")
	if err = os.WriteFile(overlayFile, overlay, 0o600); err != nil {
		return notViable
	}

	ctx, cancel := context.WithTimeout(ctx, 2*timeout+time.Minute)
	defer cancel()

	//nolint:gosec // the arguments are built from the directories given on the command line
	cmd := exec.CommandContext(ctx, "go", "test", "-race", "-count=1",
		"-timeout", timeout.String(), "-overlay", overlayFile, "./"+filepath.Base(filepath.Dir(m.file)))
	cmd.Dir = filepath.Dir(filepath.Dir(m.file))

	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out

	err = cmd.Run()

	var exit *exec.ExitError

	switch {
	case err == nil:
		return lived
	case bytes.Contains(out.Bytes(), []byte("[build failed]")), bytes.Contains(out.Bytes(), []byte("[setup failed]")):
		return notViable
	case bytes.Contains(out.Bytes(), []byte("test timed out")), ctx.Err() != nil:
		return timedOut
	case errors.As(err, &exit):
		return killed
	}

	return notViable
}

func report(all []*mutant, verbose bool) {
	type counts struct{ killed, timedOut, lived, notViable int }

	byDir := map[string]*counts{}
	total := &counts{}

	var dirs []string

	for _, m := range all {
		dir := filepath.Base(filepath.Dir(filepath.Dir(m.file))) + "/" + filepath.Base(filepath.Dir(m.file))

		c, ok := byDir[dir]
		if !ok {
			c = &counts{}
			byDir[dir] = c
			dirs = append(dirs, dir)
		}

		for _, cc := range []*counts{c, total} {
			switch m.status {
			case killed:
				cc.killed++
			case timedOut:
				cc.timedOut++
			case lived:
				cc.lived++
			default:
				cc.notViable++
			}
		}

		if verbose || m.status == lived {
			_, _ = fmt.Fprintf(os.Stdout, "%10s  %-24s %s:%d\n", m.status, m.op, dir+"/"+filepath.Base(m.file), m.pos.Line)
		}
	}

	efficacy := func(c *counts) string {
		tested := c.killed + c.timedOut + c.lived
		if tested == 0 {
			return "-"
		}

		return fmt.Sprintf("%.0f%%", 100*float64(c.killed+c.timedOut)/float64(tested))
	}

	_, _ = fmt.Fprintln(os.Stdout)

	slices.Sort(dirs)

	for _, dir := range dirs {
		c := byDir[dir]
		_, _ = fmt.Fprintf(os.Stdout, "%-30s killed %3d  timed out %3d  lived %3d  not viable %3d  efficacy %s\n",
			dir, c.killed, c.timedOut, c.lived, c.notViable, efficacy(c))
	}

	_, _ = fmt.Fprintf(os.Stdout, "%-30s killed %3d  timed out %3d  lived %3d  not viable %3d  efficacy %s\n",
		"total", total.killed, total.timedOut, total.lived, total.notViable, efficacy(total))
}
