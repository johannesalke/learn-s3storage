package main

/*
import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {

	getInput := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	presignClient := s3.NewPresignClient(s3Client)
	presignedRequest, err := presignClient.PresignGetObject(context.Background(), &getInput, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", fmt.Errorf("Error producing presigned http request")
	}
	return presignedRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil {
		return video, nil
	}
	vidURL := strings.Split(*video.VideoURL, ",")
	if len(vidURL) != 2 {
		return video, nil
	}
	bucket := vidURL[0]
	key := vidURL[1]
	newURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*5)
	if err != nil {
		return video, fmt.Errorf("%s", err)
	}
	video.VideoURL = &newURL
	return video, nil

}

*/
