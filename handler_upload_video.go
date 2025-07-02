package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

// CH3 L7
// Complete the (currently empty) handlerUploadVideo handler to store video files in S3.
// Images will stay on the local file system for now.
// I recommend using the image upload handler as a reference.
func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	// 1. Set an upload limit of 1 GB (1 << 30 bytes) using http.MaxBytesReader.
	// func MaxBytesReader(w ResponseWriter, r io.ReadCloser, n int64) io.ReadCloser
	//const uploadLimit = 1 << 30
	//limit := MaxBytesReader(w, ?, 1 << 30)
	
	// 2. Extract the videoID from the URL path parameters and parse it as a UUID
	// Copiat de handlerUploadThumbnail
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// 3. Authenticate the user to get a userID
	// Copiat de handlerUploadThumbnail
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	
	// 4. Get the video metadata from the database,
	// if the user is not the video owner, return a http.StatusUnauthorized response
	// Copiat de handlerUploadThumbnail
	videoMetadata, _ := cfg.db.GetVideo(videoID)

	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the video owner", err)
		return
	}	

	// 5. Parse the uploaded video file from the form data
	// Use (http.Request).FormFile with the key "video" to get a multipart.File in memory
	// Remember to defer closing the file with (os.File).Close - we don't want any memory leaks
	// Adaptat de handlerUploadThumbnail
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	// 6. Validate the uploaded file to ensure it's an MP4 video
	// Use mime.ParseMediaType and "video/mp4" as the MIME type
	// Adaptat de handlerUploadThumbnail
	mediatype, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mediatype != "video/mp4" || err != nil {
		respondWithError(w, http.StatusUnauthorized, "Media not valid", err)
		fmt.Printf("media not valid  : %s\n", mediatype)
		return
	}

	// 7. Save the uploaded file to a temporary file on disk.
	// Use os.CreateTemp to create a temporary file.
	// I passed in an empty string for the directory to use the system default,
	// and the name "tubely-upload.mp4" (but you can use whatever you want)
	// defer remove the temp file with os.Remove
	// defer close the temp file (defer is LIFO, so it will close before the remove)
	// io.Copy the contents over from the wire to the temp file
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to save file", err)
		return
	}
	
	fmt.Printf("Creating temp file %s\n", tempFile.Name())

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to copy file to TEMP", err)
		return
	}
 
	// CH5 L2
	// Create a processed version of the video. Upload the processed video to S3, and discard the original.
	processedFileName, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to process video ", err)
		log.Printf(err.Error())
		return
	}

	// Open processed file
	processedFile, err := os.Open(processedFileName)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to open processed video ", err)
		return
	}
	defer processedFile.Close()
	defer os.Remove(processedFile.Name())

	// Close and delete original file
	tempFile.Close()
	os.Remove(tempFile.Name())

	// CH4 L3
	aspectRatio, err := getVideoAspectRatio(processedFile.Name())
	var prefix string
	switch aspectRatio {
		case "16:9": 
			prefix = "landscape"
		case "9:16":
			prefix = "portrait"
		default:
			prefix = "other"
	} 

	// Continuem CH3 L7
	// 8. Reset the tempFile's file pointer to the beginning with .Seek(0, io.SeekStart)
	// - this will allow us to read the file again from the beginning
	// tempFile.Seek(0, io.SeekStart)
	
	fmt.Printf("Will upload %s\n", processedFile.Name())

	// 9. Put the object into S3 using PutObject. You'll need to provide:
	// The bucket name
	// The file key. Use the same <random-32-byte-hex>.ext format as the key. e.g. 1a2b3c4d5e6f7890abcd1234ef567890.mp4
	// The file contents (body). The temp file is an os.File which implements io.Reader
	// Content type, which is the MIME type of the file.
	// func (c *Client) PutObject(ctx context.Context, params *PutObjectInput, optFns ...func(*Options)) (*PutObjectOutput, error)
	key := make([]byte, 32)
	rand.Read(key)
	randomKey := base64.URLEncoding.EncodeToString(key)

	s3Key := fmt.Sprintf("%s/%s", prefix, randomKey)

	s3Params := s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &s3Key,
		Body: processedFile,
		ContentType: &mediatype,
	}

	// Retorna un PutObjectOutput que (de moment?) iognorem
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3#PutObjectOutput
	_, err = cfg.s3Client.PutObject(r.Context(), &s3Params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to copy file to S3", err)
		return
	}

	// 10. Update the VideoURL of the video record in the database with the S3 bucket and key.
	// S3 URLs are in the format https://<bucket-name>.s3.<region>.amazonaws.com/<key>.
	// Make sure you use the correct region and bucket name!
	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, s3Key)
	videoMetadata.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video ", err)
		return
	}
	fmt.Printf("Stored VideoURL  : %s\n", videoURL)





}
