package gomorph_test

import (
	"github.com/dklassen/gomorph"
	"reflect"
	"sort"
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

func double(s testSource, _ any) (testDest, error) {
	return testDest{Result: s.Value * 2}, nil
}

func triple(s testSource, _ any) (testDest, error) {
	return testDest{Result: s.Value * 3}, nil
}

func TestTransformMapSupportedOperations(t *testing.T) {
	transforms := gomorph.NewMapResolver(
		map[string]gomorph.TransformFunc[testSource, testDest, any]{
			"double": double,
			"triple": triple,
		},
	)

	mapper := gomorph.NewTransformMapper(
		transforms,
		nil, // No additional metadata needed for this example
		func(s testSource) string { return s.Op },
	)

	expectedKeys := []string{"double", "triple"}
	keys := mapper.SupportedOperations()

	sort.Strings(expectedKeys)
	sort.Strings(keys)

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("expected keys %v, got %v", expectedKeys, keys)
	}
}

func TestGenericMapMapper(t *testing.T) {
	transforms := gomorph.NewMapResolver(
		map[string]gomorph.TransformFunc[testSource, testDest, any]{
			"double": double,
			"triple": triple,
		},
	)

	mapper := gomorph.NewTransformMapper(
		transforms,
		nil, // No additional metadata needed for this example
		func(s testSource) string { return s.Op },
	)

	tests := []struct {
		name    string
		input   testSource
		want    testDest
		wantErr string
	}{
		{
			name:  "should double based on Op value",
			input: testSource{Value: 2, Op: "double"},
			want:  testDest{Result: 4},
		},
		{
			name:  "should triple based on Op value",
			input: testSource{Value: 3, Op: "triple"},
			want:  testDest{Result: 9},
		},
		{
			name:    "should raise error for no transform key",
			input:   testSource{Value: 1, Op: "unknown"},
			wantErr: "no transform for key: unknown",
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
