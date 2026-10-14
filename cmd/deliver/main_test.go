package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// arguments supplies fictional metadata only and never discovers credentials.
func arguments() []string {
	return []string{"-project-id=synthetic-project", "-project-number=123456789012", "-infisical-project=00000000-0000-4000-8000-000000000001", "-infisical-identity=00000000-0000-4000-8000-000000000002", "-source-id=00000000-0000-4000-8000-000000000003", "-source-version=7"}
}

// TestExecutionGate proves malformed/default/check commands cannot invoke the managed dependency.
func TestExecutionGate(t *testing.T) {
	for _, args := range [][]string{nil, arguments(), append(arguments(), "-execute", "-check"), append(arguments(), "-execute", "-source-version=latest"), append(arguments(), "-execute", "-secret=private"), append(arguments(), "-check")} {
		var out bytes.Buffer
		calls := 0
		status := run(context.Background(), args, &out, func(context.Context, config) error { calls++; return nil })
		if calls != 0 || strings.Contains(out.String(), "private") {
			t.Fatal("unsafe command")
		}
		if args != nil && args[len(args)-1] == "-check" && len(args) == 7 && status != 0 {
			t.Fatal("offline check failed")
		}
	}
}

// TestExecuteRedaction proves one invocation and fixed output on success and uncertain failure.
func TestExecuteRedaction(t *testing.T) {
	for _, failure := range []bool{false, true} {
		var out bytes.Buffer
		calls := 0
		result := run(context.Background(), append(arguments(), "-execute"), &out, func(ctx context.Context, c config) error {
			calls++
			if failure {
				return errors.New("private-provider-detail")
			}
			return nil
		})
		if calls != 1 || strings.Contains(out.String(), "private") || (failure && result != 1) || (!failure && result != 0) {
			t.Fatal("bad result")
		}
	}
}
