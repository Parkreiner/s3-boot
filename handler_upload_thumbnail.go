package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

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

	reqFile, reqFileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Unable to parse thumbnail payload",
			err,
		)
		return
	}
	mediaType, _, err := mime.ParseMediaType(reqFileHeader.Header.Get("Content-Type"))
	if err != nil || (mediaType != "image/jpeg" && mediaType != "image/png") {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Image must be either .png or .jpeg",
			err,
		)
		return
	}
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
	_, fileExtension, ok := strings.Cut(mediaType, "/")
	if !ok {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Image payload ",
			nil,
		)
		return
	}

	fPath := filepath.Join(cfg.assetsRoot, videoID.String()+"."+strings.ToLower(fileExtension))
	newFile, err := os.Create(fPath)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Unable to create new file",
			err,
		)
		return
	}
	defer func() {
		err := newFile.Close()
		if err != nil {
			fmt.Printf("Unable to close file %s", newFile.Name())
		}
	}()
	newFile.Chmod(0664)
	_, err = io.Copy(newFile, reqFile)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Unable to copy file contents",
			nil,
		)
	}

	clientPath := fPath
	if !strings.HasPrefix(clientPath, "/") {
		clientPath = "/" + clientPath
	}
	dbVideoDraft.ThumbnailURL = &clientPath
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
