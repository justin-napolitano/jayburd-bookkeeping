package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/providersecrets"
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
		status := run(context.Background(), args, &out, func(context.Context, config) (providersecrets.CanaryMetadata, error) {
			calls++
			return providersecrets.CanaryMetadata{}, nil
		})
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
		result := run(context.Background(), append(arguments(), "-execute"), &out, func(ctx context.Context, c config) (providersecrets.CanaryMetadata, error) {
			calls++
			if failure {
				return providersecrets.CanaryMetadata{}, errors.New("private-provider-detail")
			}
			return providersecrets.CanaryMetadata{}, nil
		})
		if calls != 1 || strings.Contains(out.String(), "private") || (failure && result != 1) || (!failure && result != 0) {
			t.Fatal("bad result")
		}
	}
}

// TestLookupWithoutID reports only selected metadata and never requires UI-only identifiers.
func TestLookupWithoutID(t *testing.T) {
	args := []string{}
	for _, arg := range arguments() {
		if !strings.HasPrefix(arg, "-source-id=") {
			args = append(args, arg)
		}
	}
	args = append(args, "-lookup")
	var out bytes.Buffer
	calls := 0
	result := run(context.Background(), args, &out, func(_ context.Context, c config) (providersecrets.CanaryMetadata, error) {
		calls++
		if !c.lookup || c.sourceID != "" {
			t.Fatal("wrong lookup mode")
		}
		return providersecrets.CanaryMetadata{SourceID: "00000000-0000-4000-8000-000000000009", SourceVersion: "7"}, nil
	})
	if result != 0 || calls != 1 || out.String() != "{\"source_id\":\"00000000-0000-4000-8000-000000000009\",\"source_version\":\"7\"}\n" {
		t.Fatal("unexpected metadata output")
	}
	out.Reset()
	if run(context.Background(), append(args, "-execute"), &out, nil) != 1 {
		t.Fatal("mixed mode accepted")
	}
}
