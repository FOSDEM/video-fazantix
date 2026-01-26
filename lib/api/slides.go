package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/fosdem/fazantix/lib/source/pdfsource"
)

// @Summary	fetch a frame from a source or sink
// @Router		/api/slides/source/{name}/set-slide/{num} [post]
// @Router		/api/slides/source/{name}/move-slide/{num} [post]
// @Tags		media
// @Param		name	path	string	true	"Name of the source to control the slides of"
// @Param		num		path	int		true	"The slide number to switch to"
// @Success	200
// @Failure	400	{string}	string	"The {name} parameter was not specified"
// @Failure	400	{string}	string	"The {num} parameter was out of range"
// @Failure	404	{string}	string	"The specified source does not exist in the configuration"
func (a *Api) handleSlidePage(w http.ResponseWriter, req *http.Request) {
	sourceName := req.PathValue("source")
	pageStr := req.PathValue("num")
	actionStr := req.PathValue("action")
	if sourceName == "" {
		http.Error(w, "Missing source name", http.StatusBadRequest)
		return
	}
	if pageStr == "" {
		http.Error(w, "Missing page number", http.StatusBadRequest)
		return
	}
	if actionStr != "set-slide" && actionStr != "move-slide" {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}
	page, err := strconv.ParseInt(pageStr, 10, 32)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var source FrameForwarderObject
	source = a.theatre.SourceByName(sourceName)
	if source == nil {
		http.Error(w, "Source does not exist", http.StatusNotFound)
		return
	}

	if req.Method != "POST" {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	relative := false
	if actionStr == "move-slide" {
		relative = true
	}

	switch src := source.(type) {
	case *pdfsource.PdfSource:
		err = src.SetPage(int(page), relative)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not set slide: %s", err), http.StatusBadRequest)
			return
		}
		return
	default:
		http.Error(w, "Unsupported source type", http.StatusBadRequest)
		return
	}
}
