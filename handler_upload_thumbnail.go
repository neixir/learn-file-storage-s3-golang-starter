package main

import (
	"path/filepath"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

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

	// 5. Get the video's metadata from the SQLite database. The apiConfig's db has a GetVideo method you can use
	videoMetadata, _ := cfg.db.GetVideo(videoID)

	// If the authenticated user is not the video owner, return a http.StatusUnauthorized response
	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the video owner", err)
		return
	}

	// CH1 L7
	// Let's update our handler to store the files on the file system. We'll save uploaded files to the /assets directory on disk.

	// 1. Instead of encoding to base64, update the handler to save the bytes to a file at the path /assets/<videoID>.<file_extension>.
	// 1.1 Use the Content-Type header to determine the file extension.
	// Per exemple "image/png"
	mediaType := header.Header.Get("Content-Type")
	file_extension := strings.Split(mediaType, "/")[1]
	
	// 1.2 Use the videoID to create a unique file path. filepath.Join and cfg.assetsRoot will be helpful here.
	filename := fmt.Sprintf("%v.%s", videoID, file_extension)
	completeFilepath := filepath.Join(cfg.assetsRoot, filename)
	fmt.Printf("filename         : %s\n", filename)
	fmt.Printf("completeFilepath : %s\n", completeFilepath)

	// 1.3 Use os.Create to create the new file
	newFile, err := os.Create(completeFilepath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create file ", err)
		return
	}

	// 1.4 Copy the contents from the multipart.File to the new file on disk using io.Copy
	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to copy file ", err)
		return
	}

	// 2. Update the thumbnail_url. Notice that in main.go we have a file server that serves files from the /assets directory.
	// The URL for the thumbnail should now be:
	// http://localhost:<port>/assets/<videoID>.<file_extension>
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	videoMetadata.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video ", err)
		return
	}

	// Restart the server and re-upload the boots-image-horizontal.png thumbnail image to ensure it's working.
	// You should see it in the UI as well as a copy in the /assets directory.

	// 8. Respond with updated JSON of the video's metadata. Use the provided respondWithJSON function and pass it the updated database.Video struct to marshal.
	respondWithJSON(w, http.StatusOK, struct{}{})
}
