package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type ffprobe struct {
	Streams	 []stream `json:"streams"`
}

type stream struct {
	Index		int `json:"index"`
	CodecType	string `json:"codec_type"`
	Width		int `json:"width"`
	Height	 	int `json:"height"`
}

// CH4 L3
// Takes a file path and returns the aspect ratio as a string.
func getVideoAspectRatio(filePath string) (string, error) {
	// It should use exec.Command to run the same ffprobe command as above.
	// In this case, the command is ffprobe and the arguments are
	// -v, error, -print_format, json, -show_streams, and the file path.
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	// Set the resulting exec.Cmd's Stdout field to a pointer to a new bytes.Buffer.
	var out bytes.Buffer
	cmd.Stdout = &out

	// .Run() the command
	err := cmd.Run()
	if err != nil {
		return "other", err
	}

	// Unmarshal the stdout of the command from the buffer's .Bytes
	// into a JSON struct so that you can get the width and height fields.
	data := ffprobe{}
	err = json.Unmarshal(out.Bytes(), &data)
	if err != nil {
		fmt.Printf("error unmarshalling ffprobe output")
		return "other", err
	}

	for _, stream := range data.Streams {
		if stream.CodecType == "video" {
			// I did a bit of math to determine the ratio,
			// then returned one of three strings: 16:9, 9:16, or other.
			tolerance := 0.1
			division := float64(stream.Width) / float64(stream.Height)

			landscape := 16.0 / 9.0
			portrait := 9.0 / 16.0

			if division > landscape-tolerance && division < landscape+tolerance {
				return "16:9", nil
			}

			if division > portrait-tolerance && division < portrait+tolerance {
				return "9:16", nil
			}
		}
	}
	
	return "other", nil

}