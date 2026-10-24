package cli

import "testing"

func TestReadSettingsFillsTheDefaults(t *testing.T) {
	t.Setenv(loginVar, "user@example.com")
	t.Setenv(hostVar, "machine.tailnet.ts.net")

	tests := []struct {
		name     string
		listen   string
		security string
		want     settings
	}{
		{
			name: "unset",
			want: settings{listen: "127.0.0.1:8700", security: "/usr/bin/security"},
		},
		{
			name:     "set",
			listen:   "127.0.0.1:8799",
			security: "/stand-in/security",
			want:     settings{listen: "127.0.0.1:8799", security: "/stand-in/security"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(listenVar, test.listen)
			t.Setenv(securityVar, test.security)

			want := test.want
			want.login, want.host = "user@example.com", "machine.tailnet.ts.net"

			if got, err := readSettings(); err != nil || got != want {
				t.Errorf("readSettings() = %+v, %v; want %+v", got, err, want)
			}
		})
	}
}
