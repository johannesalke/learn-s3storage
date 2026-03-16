package main

import (
	"encoding/base64"
	"fmt"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	//"path/filepath"
	"crypto/rand"
	"mime"
	"strings"
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

	// TODO: implement the upload here
	maxMemory := 10 << 20
	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, 400, "Error parsing form", err)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	defer file.Close()
	if err != nil {
		respondWithError(w, 400, "Error retrieving parsed file", err)
		return
	}
	//mediaType := r.Header.Get("Content-Type")

	parsedContentType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, 400, "Error parsing content type", err)
		return
	}
	if parsedContentType != "image/png" && parsedContentType != "image/jpeg" {
		respondWithError(w, 400, "Wrong file type for a thumbnail", nil)
	}

	mediaType := strings.Split(parsedContentType, "/")[1]

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 400, "Error retrieving video by ID", err)
		return
	}
	creatorID := dbVideo.CreateVideoParams.UserID
	if creatorID != userID {
		respondWithError(w, http.StatusUnauthorized, "Error: You do not have permission to edit this video.", err)
		return
	}

	//thmbnl := thumbnail{data: imgData, mediaType: mediaType}
	//videoThumbnails[videoID] = thmbnl

	//imgStr := base64.StdEncoding.EncodeToString(imgData)
	//thumbnailString := fmt.Sprintf("data:%s;base64,%s", mediaType, imgStr)
	thumbnailRNG := make([]byte, 32, 32)
	rand.Read(thumbnailRNG)
	thumbnailID := base64.RawURLEncoding.EncodeToString(thumbnailRNG)

	thumbnailFilepath := fmt.Sprintf("%s/%s.%s", cfg.assetsRoot, thumbnailID, mediaType)
	filePtr, err := os.Create(thumbnailFilepath)
	if err != nil {
		respondWithError(w, 400, "Error creating thumbnail file on disk", err)
	}
	io.Copy(filePtr, file)

	//var thumbnailURL *string
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, thumbnailID, mediaType)
	dbVideo.ThumbnailURL = &thumbnailURL
	cfg.db.UpdateVideo(dbVideo)

	respondWithJSON(w, http.StatusOK, dbVideo)
}
