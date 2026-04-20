package validator

import "testing"

func TestValidatePortRange(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		min     int
		max     int
		wantErr bool
	}{
		{
			name:    "valid port in range",
			port:    8080,
			min:     1024,
			max:     65535,
			wantErr: false,
		},
		{
			name:    "port below minimum",
			port:    1023,
			min:     1024,
			max:     65535,
			wantErr: true,
		},
		{
			name:    "port above maximum",
			port:    65536,
			min:     1024,
			max:     65535,
			wantErr: true,
		},
		{
			name:    "port at minimum boundary",
			port:    1024,
			min:     1024,
			max:     65535,
			wantErr: false,
		},
		{
			name:    "port at maximum boundary",
			port:    65535,
			min:     1024,
			max:     65535,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePortRange(tt.port, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePortRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{
			name:    "non-empty value",
			field:   "username",
			value:   "admin",
			wantErr: false,
		},
		{
			name:    "empty value",
			field:   "password",
			value:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			field:   "name",
			value:   "   ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequired() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAuthMethod(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{
			name:    "valid password method",
			method:  "password",
			wantErr: false,
		},
		{
			name:    "valid private_key method",
			method:  "private_key",
			wantErr: false,
		},
		{
			name:    "invalid method",
			method:  "oauth",
			wantErr: true,
		},
		{
			name:    "empty method",
			method:  "",
			wantErr: true,
		},
		{
			name:    "case sensitive check",
			method:  "Password",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuthMethod(tt.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAuthMethod() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		wantErr  bool
	}{
		{
			name:     "valid round_robin",
			strategy: "round_robin",
			wantErr:  false,
		},
		{
			name:     "valid least_rules",
			strategy: "least_rules",
			wantErr:  false,
		},
		{
			name:     "valid weighted",
			strategy: "weighted",
			wantErr:  false,
		},
		{
			name:     "invalid strategy",
			strategy: "random",
			wantErr:  true,
		},
		{
			name:     "empty strategy",
			strategy: "",
			wantErr:  true,
		},
		{
			name:     "case sensitive check",
			strategy: "Round_Robin",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStrategy(tt.strategy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStrategy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
