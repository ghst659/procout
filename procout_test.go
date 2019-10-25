package procout

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestProcOutsErrs(t *testing.T) {
	ctx := context.Background()
	fake, err := makeFake("/tmp",
		`#!/bin/bash
declare -a -r argv=( "$@" )
for (( i=0; i < "${#argv[@]}"; i++ )); do
    if (( i % 2 == 0)) ; then
        echo even "${i}" "${argv[$i]}" >&1
    else
        echo odd "${i}" "${argv[$i]}" >&2
    fi
done
exit 0
`)
	if err != nil {
		t.Fatalf("error creating file: %v", err)
	}
	defer os.Remove(fake)
	outs, errs, err := ProcOutsErrs(ctx, fake, "zero", "one", "two")
	if err != nil {
		t.Errorf("%s: Run error: %v", fake, err)
	}
	var gotOut strings.Builder
	var gotErr strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)
	go getLines(t, outs, &wg, &gotOut)
	go getLines(t, errs, &wg, &gotErr)
	wg.Wait()
	if gotOut.String() != "even 0 zero|even 2 two|" {
		t.Errorf("failed stdout: %q", gotOut.String())
	}
	if gotErr.String() != "odd 1 one|" {
		t.Errorf("failed stderr: %q", gotErr.String())
	}
}

func getLines(t *testing.T, ch <-chan string, wg *sync.WaitGroup, buf *strings.Builder) {
	defer wg.Done()
	for line := range ch {
		if _, err := buf.WriteString(line); err != nil {
			t.Logf("error writing buffer: %v", err)
			return
		}
		buf.WriteRune('|')
	}
}

func makeFake(directory, innards string) (string, error) {
	f, err := ioutil.TempFile(directory, "fake*.sh")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(innards); err != nil {
		return "", err
	}
	if err := f.Chmod(0777); err != nil {
		return "", err
	}
	return f.Name(), nil
}
