package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestDouyinCookieFlag(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   string
		set     bool
		wantErr bool
	}{
		{"omitted", "", false, false},
		{"provided", "UIFID=test; ttwid=test", true, false},
		{"empty", "  ", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("cookie", "", "")
			if tc.set {
				if err := cmd.Flags().Set("cookie", tc.value); err != nil {
					t.Fatal(err)
				}
			}
			got, err := douyinCookieFlag(cmd)
			if (err != nil) != tc.wantErr || (!tc.wantErr && got != tc.value) {
				t.Fatalf("cookie flag error = %v, want error %v", err, tc.wantErr)
			}
		})
	}
}
