package main

import (
	"encoding/base64"
	"fmt"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"io"
	"net/http"
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
	if err != nil {
		respondWithError(w, 400, "Error retrieving parsed file", err)
		return
	}
	//mediaType := r.Header.Get("Content-Type")
	mediaType := fileHeader.Header.Get("Content-Type")

	var imgData []byte
	imgData, err = io.ReadAll(file)
	if err != nil {
		respondWithError(w, 400, "Error reading img data", err)
		return
	}

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

	thmbnl := thumbnail{data: imgData, mediaType: mediaType}
	videoThumbnails[videoID] = thmbnl

	//var thumbnailURL *string
	thumbnailURL := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoID)
	dbVideo.ThumbnailURL = &thumbnailURL
	cfg.db.UpdateVideo(dbVideo)

	respondWithJSON(w, http.StatusOK, dbVideo)
}
