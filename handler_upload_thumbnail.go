package main

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory10Megabytes = 10 << 20

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

	dbVideoDraft, err := cfg.db.GetVideo(videoID)
	if err == sql.ErrNoRows {
		respondWithError(
			w,
			http.StatusNotFound,
			fmt.Sprintf("Unable to find video with ID %s", videoID),
			err,
		)
		return
	}

	err = r.ParseMultipartForm(maxMemory10Megabytes)
	if err != nil {
		log.Printf("Unable to parse multi-part form with memory budget %d\n", maxMemory10Megabytes)
		respondWithError(
			w,
			http.StatusBadRequest,
			"Unable to parse thumbnail request",
			err,
		)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Unable to parse thumbnail payload",
			err,
		)
		return
	}
	mediaType := fileHeader.Header.Get("Content-Type")
	if mediaType == "" {
		msg := "Thumbnail is missing media type"
		respondWithError(
			w,
			http.StatusBadRequest,
			msg,
			errors.New(msg),
		)
		return
	}
	bytes, err := io.ReadAll(file)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Unable to store thumbnail",
			err,
		)
		return
	}

	dataUrl := "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(bytes)
	dbVideoDraft.ThumbnailURL = &dataUrl
	dbVideoDraft.UpdatedAt = time.Now()
	err = cfg.db.UpdateVideo(dbVideoDraft)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Unable to update thumbnail data",
			err,
		)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideoDraft)
}
