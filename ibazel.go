package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lukegb/ibazel/depresolver"
	"golang.org/x/exp/inotify"
)

const (
	bazelBinFlag = "--bazel="
)

var (
	goodCommands = map[string]bool{"test": true, "build": true, "run": true}
	writeDelay   = 100 * time.Millisecond
)

type Resolver interface {
	Resolve(ctx context.Context, target string) (sourceFiles, buildFiles []string, err error)
}

type ibazel struct {
	Resolver Resolver

	sourceWatcher *inotify.Watcher
	sourceFiles   []string
	buildWatcher  *inotify.Watcher
	buildFiles    []string

	rebuildChan chan int

	BazelBin  string
	BazelArgs []string

	Command string
	Targets []string
}

func (i *ibazel) setupWatcher(ctx context.Context, watchFiles []string, w **inotify.Watcher) error {
	if *w != nil {
		if err := (*w).Close(); err != nil {
			return err
		}
	}

	var err error
	if *w, err = inotify.NewWatcher(); err != nil {
		return err
	}

	for _, f := range watchFiles {
		if err := (*w).AddWatch(f, inotify.IN_CLOSE_WRITE); err != nil {
			return err
		}
	}

	return nil
}

func (i *ibazel) computeFileSet(ctx context.Context) error {
	targetStr := strings.Join(i.Targets, " ")
	var err error
	i.sourceFiles, i.buildFiles, err = i.Resolver.Resolve(ctx, targetStr)
	return err
}

func (i *ibazel) rewatch(ctx context.Context) error {
	if err := i.setupWatcher(ctx, i.sourceFiles, &i.sourceWatcher); err != nil {
		return err
	}
	if err := i.setupWatcher(ctx, i.buildFiles, &i.buildWatcher); err != nil {
		return err
	}

	return nil
}

func (i *ibazel) FourTwenty(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, i.BazelBin, append(i.BazelArgs, append([]string{i.Command}, i.Targets...)...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err, ok := err.(*exec.ExitError); ok {
		fmt.Printf("bazel failed [:'(]: %s\n", err)
		// *shrug*
		return nil
	}
	return err
}

func (i *ibazel) Run(ctx context.Context) error {
rewatch:
	for {
		log.Println("Computing set of files to watch...")
		if err := i.computeFileSet(ctx); err != nil {
			return err
		}

	rebuild:
		for {
			log.Println("Rebuilding!")
			if err := i.FourTwenty(ctx); err != nil {
				return err
			}

			if err := i.rewatch(ctx); err != nil {
				return err
			}
			log.Println("...waiting for next event")
			select {
			case <-i.sourceWatcher.Event:
				// source has changed
				time.Sleep(writeDelay)
				continue rebuild
			case <-i.buildWatcher.Event:
				// BUILD has changed!
				time.Sleep(writeDelay)
				continue rewatch
			}
		}
	}
	panic("wat")
}

func usage() {
	fmt.Fprintf(os.Stderr, `%s [flags...] command target...

All flags are passed literally to bazel, unless the first flag is --bazel=, which can be used to specify the path to your bazel binary.
`, os.Args[0])
}

func main() {
	ctx := context.Background()

	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	}
	args := os.Args[1:]
	bazelBin := "bazel"

	if strings.HasPrefix(args[0], bazelBinFlag) {
		bazelBin = strings.TrimPrefix(args[0], bazelBinFlag)
		args = args[1:]
	}

	var targets []string
	var flags []string

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--"):
			flags = append(flags, arg)
		default:
			targets = append(targets, arg)
		}
	}

	if len(targets) < 2 {
		usage()
		os.Exit(1)
	}

	cmd, targets := targets[0], targets[1:]

	if !goodCommands[cmd] {
		fmt.Fprintf(os.Stderr, "%q is not a valid command for use with ibazel - try 'build', 'test', or 'run'.\n\n", cmd)
		usage()
		os.Exit(1)
	}

	dr := &depresolver.CommandDepResolver{
		BazelBin:  bazelBin,
		BazelArgs: flags,
	}
	ib := &ibazel{
		Resolver:  dr,
		BazelBin:  bazelBin,
		BazelArgs: flags,
		Command:   cmd,
		Targets:   targets,
	}
	panic(ib.Run(ctx))
}
