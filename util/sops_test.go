package util

import (
	"testing"
)

func TestGetFileFormat(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "yaml",
			args: args{
				filename: "/path/to/test.yaml",
			},
			want: "yaml",
		},
		{
			name: "json",
			args: args{
				filename: "/path/to/test.json",
			},
			want: "json",
		},
		{
			name: "ini",
			args: args{
				filename: "/path/to/test.ini",
			},
			want: "ini",
		},
		{
			name: "dotenv extension",
			args: args{
				filename: "/path/to/test.env",
			},
			want: "dotenv",
		},
		{
			name: "dotenv alone",
			args: args{
				filename: "/path/to/.env",
			},
			want: "dotenv",
		},
		{
			name: "binary",
			args: args{
				filename: "/path/to/test.whatever",
			},
			want: "binary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFileFormat(tt.args.filename); got != tt.want {
				t.Errorf("getFileFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}
