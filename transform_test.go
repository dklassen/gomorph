package gomorph_test

import (
	"github.com/dklassen/gomorph"
	"reflect"
	"strings"
	"testing"
)

type testSource struct {
	Value int
	Op    string
}

type testDest struct {
	Result int
}

func double(s testSource) testDest {
	return testDest{Result: s.Value * 2}
}

func triple(s testSource) testDest {
	return testDest{Result: s.Value * 3}
}

func TestGenericMapMapper(t *testing.T) {
	transforms := gomorph.TransformMap[string, testSource, testDest]{
		"double": double,
		"triple": triple,
	}
	mapper := gomorph.NewTransformMapper(
		transforms,
		func(s testSource) string { return s.Op },
	)

	tests := []struct {
		name    string
		input   any
		want    testDest
		wantErr string // empty means no error expected
	}{
		{
			"should double based on Op value",
			testSource{Value: 2, Op: "double"},
			testDest{Result: 4},
			"",
		},
		{
			"should triple based on Op value",
			testSource{Value: 3, Op: "triple"},
			testDest{Result: 9},
			"",
		},
		{
			"should raise error for invalid type",
			"not a testSource",
			testDest{},
			"expected gomorph_test.testSource, got string",
		},
		{
			"should raise error for no transform key",
			testSource{Value: 1, Op: "unknown"},
			testDest{},
			"no transform for key: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapper.From(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
			}
		})
	}
}
