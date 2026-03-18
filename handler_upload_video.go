package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	//"time"

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
		respondWithError(w, 400, "Wrong file type for a video", err)
		return
	}

	//Read the video data into a temprary file, which can later be uploaded to S3 storage bucket.
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, 400, "Error creating temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	io.Copy(tempFile, formFile)

	tempFile.Seek(0, io.SeekStart)

	vidRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, 400, "Error in aspect ratio determination:", err)
		return
	}

	vidRNG := make([]byte, 32, 32)
	rand.Read(vidRNG)
	vidKey := fmt.Sprintf("%s/%x.mp4", vidRatio, vidRNG)

	//Create a faststarting version of the video based on the initial temp file.
	fastloadFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, 400, "Error in conversion to fast loading:", err)
		return
	}
	fastloadFile, err := os.Open(fastloadFilePath)
	if err != nil {
		respondWithError(w, 400, "Error opening fastloading temp file:", err)
		return
	}
	defer fastloadFile.Close()

	testfile, _ := os.OpenFile("/tmp/testvid.mp4", os.O_WRONLY|os.O_CREATE, 0644)
	io.Copy(testfile, fastloadFile)
	/*cmd := exec.Command("head", "-c", "1000", fastloadFile.Name(), ">", "strings.txt")
	err = cmd.Run()
	if err != nil {
		respondWithError(w, 400, "Checking if file starts with moov", err)
		return
	}*/
	fastloadFile.Seek(0, io.SeekStart)

	objectInput := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &vidKey,
		Body:        fastloadFile,
		ContentType: &parsedContentType,
	}

	_, err = cfg.s3Client.PutObject(context.Background(), &objectInput)
	if err != nil {
		respondWithError(w, 400, "Error uploading file:", err)
		return
	}

	//Update db video object so it points at the new aws location of the video.
	//videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, vidKey)
	videoURL := cfg.s3Bucket + "," + vidKey
	dbVideo.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, 400, "Error updating db", err)
		return
	}

	video, err := cfg.dbVideoToSignedVideo(dbVideo)
	if err != nil {
		respondWithError(w, 400, "", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
	return
}

func getVideoAspectRatio(filePath string) (string, error) {
	//commandStr := fmt.Sprintf()
	//fmt.Println(filePath)
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	vidData := new(bytes.Buffer)
	cmd.Stdout = vidData
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error running vid eval command:%s", err)
	}
	var result struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	//bytes := vidData.Bytes()
	err = json.Unmarshal(vidData.Bytes(), &result)
	if err != nil {
		return "", fmt.Errorf("Error unmarshalling json:")
	}
	if result.Streams[0].Width/result.Streams[0].Height == 16/9 {
		return "landscape", nil
	} else if result.Streams[0].Width/result.Streams[0].Height == 9/16 {
		return "portrait", nil
	} else {
		return "other", nil
	}

}

func processVideoForFastStart(filePath string) (string, error) {
	newFilePath := filePath + ".processing.mp4"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newFilePath)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error running vid eval command:%s", err)
	}
	return newFilePath, nil
}
