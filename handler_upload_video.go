package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	const maxSize = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	userID, err := validateUser(r, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization error", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to find video in database", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You must be the author of the video", nil)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse video", err)
		return
	}

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}

	if mediaType != "video/mp4"  {
		respondWithError(w, http.StatusBadRequest, "Wrong media type, must be mp4", nil)
		return
	}

	fileExtension := strings.Split(mediaType, "/")[1]
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	key :=  base64.RawURLEncoding.EncodeToString(randBytes) + "." + fileExtension

	tempFile, err := os.CreateTemp("", "tubely-upload-*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create video file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy video file", err)
		return
	}

	tempFile.Seek(0, io.SeekStart)


	_, err = cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key:	&key,
		Body:	tempFile,
		ContentType: &mediaType,
	})
	if err != nil{
		respondWithError(w, http.StatusInternalServerError, "Unable to upload video file", err)
		return
	}

	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)

	video.VideoURL = &videoURL

	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}
