package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

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


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// CH1 L05
	// 1. Authentication has already been taken care of for you, and the video's ID has been parsed from the URL path.
	// 2. Parse the form 
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// 3. Get the image data from the form
	// 3.1 Use r.FormFile to get the file data and file headers. The key the web browser is using is called "thumbnail"
	// func (r *Request) FormFile(key string) (multipart.File, *multipart.FileHeader, error)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	// 3.2 Get the media type from the form file's Content-Type header
	mediaType := header.Header.Get("Content-Type")

	// 4. Read all the image data into a byte slice using io.ReadAll
	imageData, err := io.ReadAll(file)

	// 5. Get the video's metadata from the SQLite database. The apiConfig's db has a GetVideo method you can use
	videoMetadata, _ := cfg.db.GetVideo(videoID)

	// If the authenticated user is not the video owner, return a http.StatusUnauthorized response
	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the video owner", err)
		return
	}

	// CH1 L6
	base64ImageData := base64.StdEncoding.EncodeToString(imageData)
	thumbnailURL := fmt.Sprintf("data:%s;base64,%s", mediaType, base64ImageData)
	videoMetadata.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video ", err)
		return
	}

	// 8. Respond with updated JSON of the video's metadata. Use the provided respondWithJSON function and pass it the updated database.Video struct to marshal.
	respondWithJSON(w, http.StatusOK, struct{}{})

	// 9. Test your handler manually by using the Tubely UI to upload the boots-image-horizontal.png image. You should see the thumbnail update in the UI!
}
