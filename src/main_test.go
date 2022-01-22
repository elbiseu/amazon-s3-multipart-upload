package main

import "testing"

func TestGetFilenameExtension(t *testing.T) {
	type args struct {
		contentType string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should get the file name extension for the content type image/gif",
			args: args{
				contentType: "image/gif",
			},
			want: ".gif",
		},
		{
			name: "should get the file name extension for the content type image/jpeg",
			args: args{
				contentType: "image/jpeg",
			},
			want: ".jpeg",
		},
		{
			name: "should get the file name extension for the content type image/png",
			args: args{
				contentType: "image/png",
			},
			want: ".png",
		},
		{
			name: "should get the file name extension for the content type image/tiff",
			args: args{
				contentType: "image/tiff",
			},
			want: ".tiff",
		},
		{
			name: "should get the file name extension for the content type video/quicktime",
			args: args{
				contentType: "video/quicktime",
			},
			want: ".mov",
		},
		{
			name: "should get the file name extension for the content type video/mpeg",
			args: args{
				contentType: "video/mpeg",
			},
			want: ".mpeg",
		},
		{
			name: "should get the file name extension for the content type video/mp4",
			args: args{
				contentType: "video/mp4",
			},
			want: ".mp4",
		},
		{
			name: "should get the file name extension for the content type video/webm",
			args: args{
				contentType: "video/webm",
			},
			want: ".webm",
		},
	}
	t.Run("get filename extension", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := GetFilenameExtension(tt.args.contentType); got != tt.want {
					t.Errorf("GetFilenameExtension() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}
