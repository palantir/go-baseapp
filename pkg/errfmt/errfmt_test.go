// Copyright 2022 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errfmt

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrint(t *testing.T) {
	t.Run("nilError", func(t *testing.T) {
		assert.Empty(t, Print(nil), "nil error did not product empty output")
	})

	t.Run("plainError", func(t *testing.T) {
		err := errors.New("this is an error")
		assert.Equal(t, "this is an error", Print(err), "incorrect error output")
	})

	t.Run("nestedError", func(t *testing.T) {
		root := errors.New("this is an error")
		err := pkgerrors.WithMessage(root, "context 1")
		err = pkgerrors.WithMessage(err, "context 2")
		err = pkgerrors.WithMessage(err, "context 3")
		assert.Equal(t, "context 3: context 2: context 1: this is an error", Print(err), "incorrect error output")
	})

	t.Run("pkgErrorsStackTrace", func(t *testing.T) {
		const depth = 3
		const minLines = 1 + 2*(depth+1)

		err := recursiveError(
			depth,
			func() error { return errors.New("this is an error") },
			func(err error) error { return pkgerrors.Wrap(err, "context") },
		)

		out := Print(err)
		t.Log(out)

		outLines := strings.Split(out, "\n")
		require.True(t, len(outLines) > minLines, "expected at least %d error lines, but got %d", minLines, len(outLines))

		assert.Equal(t, "context: context: context: this is an error", outLines[0], "incorrect error message")
		assert.Contains(t, outLines[3], "errfmt.recursiveError", "incorrect stack trace")
		assert.Contains(t, outLines[5], "errfmt.recursiveError", "incorrect stack trace")
		assert.Contains(t, outLines[7], "errfmt.recursiveError", "incorrect stack trace")
	})

	t.Run("runtimeStackTrace", func(t *testing.T) {
		const depth = 3
		const minLines = 1 + 2*(depth+1)

		err := recursiveError(
			depth,
			func() error { return newStackTraceError("this is an error") },
			func(err error) error { return err },
		)

		out := Print(err)
		t.Log(out)

		outLines := strings.Split(out, "\n")
		require.True(t, len(outLines) > minLines, "expected at least %d error lines, but got %d", minLines, len(outLines))

		assert.Equal(t, "this is an error", outLines[0], "incorrect error message")
		assert.Contains(t, outLines[3], "errfmt.recursiveError", "incorrect stack trace")
		assert.Contains(t, outLines[5], "errfmt.recursiveError", "incorrect stack trace")
		assert.Contains(t, outLines[7], "errfmt.recursiveError", "incorrect stack trace")
	})
}

func recursiveError(depth int, root func() error, wrap func(error) error) error {
	if depth == 0 {
		return root()
	}
	return wrap(recursiveError(depth-1, root, wrap))
}

type stackTraceError struct {
	msg string
	st  []runtime.Frame
}

func newStackTraceError(msg string) error {
	callers := make([]uintptr, 32)

	n := runtime.Callers(2, callers)
	frames := runtime.CallersFrames(callers[0:n])

	var stack []runtime.Frame
	for {
		f, more := frames.Next()
		if !more {
			break
		}
		stack = append(stack, f)
	}

	return stackTraceError{msg: msg, st: stack}
}

func (ste stackTraceError) Error() string               { return ste.msg }
func (ste stackTraceError) StackTrace() []runtime.Frame { return ste.st }
