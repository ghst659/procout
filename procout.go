// Package procout provides functions to run subprograms
// and stream their stdout and stderr.
package procout

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
)

func ProcOutsErrs(ctx context.Context, line ...string) (<-chan string, <-chan string, error) {
	if len(line) == 0 {
		return nil, nil, fmt.Errorf("null command line")
	}
	cmd := exec.CommandContext(ctx, line[0], line[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	var wg sync.WaitGroup
	outs := make(chan string)
	errs := make(chan string)
	streamCtx, streamCancel := context.WithCancel(ctx)
	wg.Add(2)
	go streamLines(streamCtx, stdout, &wg, outs)
	go streamLines(streamCtx, stderr, &wg, errs)
	go func() {
		defer streamCancel()
		wg.Wait()
		if err := cmd.Wait(); err != nil {
			log.Printf("%v", err)
		}
	}()
	return outs, errs, nil
}

func streamLines(ctx context.Context, reader io.Reader, wg *sync.WaitGroup, ch chan<- string) {
	defer wg.Done()
	defer close(ch)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		text := scanner.Text()
		select {
		case ch <- text:
		case <-ctx.Done():
			return
		}
	}
}
