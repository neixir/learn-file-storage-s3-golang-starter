package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

// CH6 L6 (Step 5)
// It should take a video database.Video as input and return a database.Video with the VideoURL field set
// to a presigned URL and an error (to be returned from the handler)
func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	
	if video.VideoURL == nil {
		return video, fmt.Errorf("No bucket/key.")
	}

	// It should first split the video.VideoURL on the comma to get the bucket and key
	sep := strings.Split(*video.VideoURL, ",")
	if len(sep) != 2 {
		return video, fmt.Errorf("No bucket/key.")
	}
		
	bucket := sep[0]
	key := sep[1]

	// Then it should use generatePresignedURL to get a presigned URL for the video
	presignedUrl, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Duration(5 * time.Minute))
	if err != nil {
		return video, err
	}
	
	// Set the VideoURL field of the video to the presigned URL and return the updated video
	video.VideoURL = &presignedUrl

	log.Printf("Stored VideoURL  : %s (dbVideoToSignedVideo)\n", *video.VideoURL)

	return video, nil

}
