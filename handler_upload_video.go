package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	//Authenticate User
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
	//Check if User has authority to edit video
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

	//Parse form and load video data into memory.
	formFile, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, 400, "Couldn't parse form files", err)
		return
	}
	defer formFile.Close()

	//Check that the filetype is correct.
	parsedContentType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, 400, "Error parsing content type", err)
		return
	}
	if parsedContentType != "video/mp4" {
		respondWithError(w, 400, "Wrong file type for a video", nil)
	}

	//Read the video data into a temprary file, which can later be uploaded to S3 storage bucket.
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, 400, "Error creating temp file", nil)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	io.Copy(tempFile, formFile)

	tempFile.Seek(0, io.SeekStart)

	vidRNG := make([]byte, 32, 32)
	rand.Read(vidRNG)
	vidKey := fmt.Sprintf("%x.mp4", vidRNG)

	objectInput := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &vidKey,
		Body:        tempFile,
		ContentType: &parsedContentType,
	}

	cfg.s3Client.PutObject(context.Background(), &objectInput)

	//Update db video object so it points at the new aws location of the video.
	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, vidKey)
	dbVideo.VideoURL = &videoURL
	cfg.db.UpdateVideo(dbVideo)

}
