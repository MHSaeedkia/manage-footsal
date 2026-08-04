package bot

import "testing"

func TestAPIEndpointFor(t *testing.T) {
	tests := []struct {
		platform string
		want     string
		wantErr  bool
	}{
		{PlatformBale, "https://tapi.bale.ai/bot%s/%s", false},
		{PlatformTelegram, "https://api.telegram.org/bot%s/%s", false},
		{"", "", true},
		{"Bale", "", true},
		{"whatsapp", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			got, err := APIEndpointFor(tt.platform)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("APIEndpointFor(%q) expected an error, got endpoint %q", tt.platform, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("APIEndpointFor(%q) unexpected error: %v", tt.platform, err)
			}
			if got != tt.want {
				t.Errorf("APIEndpointFor(%q) = %q, want %q", tt.platform, got, tt.want)
			}
		})
	}
}
