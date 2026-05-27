package bootmode

import "testing"

func TestParseKernelCmdline(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cmdline string
		want    string
	}{
		{name: "web flag", cmdline: "quiet splash rdos.ui=web", want: ModeWeb},
		{name: "legacy flag", cmdline: "rdos.ui=legacy loglevel=3", want: ModeLegacy},
		{name: "no flag", cmdline: "quiet loglevel=3", want: ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseKernelCmdline(tc.cmdline); got != tc.want {
				t.Fatalf("ParseKernelCmdline() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	if got := Resolve("quiet rdos.ui=legacy", ModeWeb); got != ModeLegacy {
		t.Fatalf("Resolve cmdline override = %q, want legacy", got)
	}

	if got := Resolve("quiet", ModeLegacy); got != ModeLegacy {
		t.Fatalf("Resolve default legacy = %q, want legacy", got)
	}

	if got := Resolve("quiet", ""); got != ModeWeb {
		t.Fatalf("Resolve default web = %q, want web", got)
	}
}
