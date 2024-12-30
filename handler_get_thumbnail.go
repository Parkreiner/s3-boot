package main

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/google/uuid"
)

var mediaTypeRe = regexp.MustCompile(`data:image\/([a-zA-Z]+);base64,\w+={0,2}`)

func (cfg *apiConfig) handlerThumbnailGet(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Thumbnail not found", nil)
		return
	}
	if dbVideo.ThumbnailURL == nil {
		respondWithError(w, http.StatusNotFound, "Video does not have thumbnail", nil)
		return
	}

	mediaType := mediaTypeRe.Find([]byte(*dbVideo.ThumbnailURL))
	if mediaType == nil {
		respondWithError(w, http.StatusInternalServerError, "Data URL does not have encoded media type", nil)
		return
	}

	w.Header().Set("Content-Type", string(mediaType))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(*dbVideo.ThumbnailURL)))

	_, err = w.Write([]byte(*dbVideo.ThumbnailURL))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error writing response", err)
		return
	}
}
