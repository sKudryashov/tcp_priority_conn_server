package formatter

import (
	"reflect"
	"testing"
)

func TestBit_ParseRequest(t *testing.T) {
	type args struct {
		body []byte
	}
	tests := []struct {
		name        string
		payload     int
		action      string
		payloadSize int
		args        args
		wantErr     bool
	}{
		{
			name: "req test",
			args: args{
				body: []byte("\bcOpIh4bS"),
			},
			action:      "0",
			payloadSize: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, payloadSize, err := ParseRequest(tt.args.body[0])
			if (err != nil) != tt.wantErr {
				t.Errorf("Bit.ParseRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if action != tt.action {
				t.Fatalf("GetAction() != tt.action")
			}
			if payloadSize != int64(tt.payloadSize) {
				t.Fatalf("b.GetPayloadSize() != tt.payloadSize")
			}
		})
	}
}

func TestFormatPopResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		output  []byte
		wantErr bool
	}{
		{
			input:   []byte("4N+61Z7F"),
			output:  []byte{8, 52, 78, 43, 54, 49, 90, 55, 70},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := FormatPopResponse(tt.input)
			if !reflect.DeepEqual(response, tt.output) {
				t.Fatalf(" unexpected output %s %s expected", string(response), string(tt.output))
			}
		})
	}
}
