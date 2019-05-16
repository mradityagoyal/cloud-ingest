package copy

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestNewSemAcquiringReader(t *testing.T) {
	tests := []struct {
		concurrentReadMax int
		want              string
	}{
		{-1, "*strings.Reader"},
		{0, "*copy.SemAcquiringReader"},
		{1, "*copy.SemAcquiringReader"},
	}
	for _, tc := range tests {
		ctx := context.Background()
		var stringsReader io.Reader = strings.NewReader("some input")
		*concurrentReadMax = tc.concurrentReadMax
		r := NewSemAcquiringReader(stringsReader, ctx)
		got := reflect.TypeOf(r).String()
		if got != tc.want {
			t.Errorf("concurrentReadMax = %v, NewSemAcquiringReader got type %s, want %s", tc.concurrentReadMax, got, tc.want)
		}
	}

}
