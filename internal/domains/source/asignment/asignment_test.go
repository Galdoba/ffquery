package asignment_test

import (
	"github.com/Galdoba/ffquery/internal/domains/source/asignment"
	"testing"
)

func TestGetFileNames(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		dir     string
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := asignment.GetFileNames(tt.dir)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("GetFileNames() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("GetFileNames() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("GetFileNames() = %v, want %v", got, tt.want)
			}
		})
	}
}
