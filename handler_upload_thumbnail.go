package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	userID, err := validateUser(r, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization error", err)
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse video", err)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
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

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Wrong media type, must be jpeg or png", nil)
		return
	}

	fileExtension := strings.Split(mediaType, "/")[1]
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	videoWithExtension :=  base64.RawURLEncoding.EncodeToString(randBytes) + "." + fileExtension

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, videoWithExtension)

	videoFilePath := filepath.Join(cfg.assetsRoot, videoWithExtension)

	assetsFile, err := os.Create(videoFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create video file", err)
		return
	}
	defer assetsFile.Close()

	if _, err := io.Copy(assetsFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save file", err)
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
	
	video.ThumbnailURL = &thumbnailURL

	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}
