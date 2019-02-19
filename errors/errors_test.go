package errors

import (
	"io"
	"testing"
)

func TestIs(t *testing.T) {
	root := New("root")
	child := Wrap(root, "child")
	grandchild := Wrap(child, "grandchild")

	cases := map[string]struct {
		cause  *Error
		err    error
		wantIs bool
	}{
		"root itself": {
			cause:  root,
			err:    root,
			wantIs: true,
		},
		"grandchild itself": {
			cause:  grandchild,
			err:    grandchild,
			wantIs: true,
		},
		"root and child": {
			cause:  root,
			err:    child,
			wantIs: true,
		},
		"root and grandchild": {
			cause:  root,
			err:    grandchild,
			wantIs: true,
		},
		"child and grandchild": {
			cause:  child,
			err:    grandchild,
			wantIs: true,
		},
		"child and root": {
			cause:  child,
			err:    root,
			wantIs: false,
		},
		"not related with grandchild": {
			cause:  New("foo"),
			err:    grandchild,
			wantIs: false,
		},
		"grandchild with not related": {
			cause:  grandchild,
			err:    New("foo"),
			wantIs: false,
		},
		"external error is itself": {
			cause:  ErrExternal,
			err:    ErrExternal,
			wantIs: true,
		},
		"std library error is external": {
			cause:  ErrExternal,
			err:    io.EOF,
			wantIs: true,
		},
		"nil is nil": {
			cause:  nil,
			err:    nil,
			wantIs: true,
		},
		"stdlib is not nil": {
			cause:  nil,
			err:    io.EOF,
			wantIs: false,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if got := tc.cause.Is(tc.err); got != tc.wantIs {
				t.Fatal("unexpected result")
			}
		})
	}
}
