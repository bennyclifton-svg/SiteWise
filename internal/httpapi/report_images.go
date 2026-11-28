package httpapi

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // a report image may be a JPEG
	_ "image/png"
	"io"
	"net/http"
	"strings"

	"sitewise/internal/store"
)

// Images in the project report's cells (project report plan, slice 5;
// Lane C). The upload is decoded, not trusted by its name or type; PNG and
// JPEG only, at most 8 MB; served only through the report that holds it,
// to the report's own org. Nothing here calls Jev or the model.

func init() {
	routePaths["POST /reports/{r}/images"] = pathReportWrite
	routePaths["PUT /reports/{r}/images/place"] = pathReportWrite
	routePaths["GET /reports/{r}/images/{sha}"] = pathReportRead
	noOperation["POST /reports/{r}/images"] = "a file upload; chat does not receive files (reports.place_image puts a filed image in a cell)"
	for pattern, h := range map[string]func(http.ResponseWriter, *http.Request, Deps){
		"POST /reports/{r}/images":      postReportImage,
		"PUT /reports/{r}/images/place": placeReportImage,
		"GET /reports/{r}/images/{sha}": getReportImage,
	} {
		reportRowRoutes[pattern] = h
	}
}

// imageReply is a saved image and the line that shows it in a cell.
type imageReply struct {
	SHA256 string `json:"sha256"`
	Line   string `json:"line"`
}

func imageLine(caption, sum string) string {
	caption = strings.NewReplacer("]", "", "[", "", "\n", " ").Replace(strings.TrimSpace(caption))
	if len(caption) > 200 {
		caption = caption[:200]
	}
	return fmt.Sprintf("![%s](image:%s)", caption, sum)
}

// postReportImage saves one dropped image: the body is the file's bytes,
// the caption a query value.
func postReportImage(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	if deps.Blobs == nil {
		http.Error(w, "file storage unavailable", http.StatusServiceUnavailable)
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, store.ReportImageMaxBytes))
	if err != nil {
		http.Error(w, "An image is at most 8 MB.", http.StatusRequestEntityTooLarge)
		return
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 12000 || cfg.Height > 12000 {
		http.Error(w, "Only a PNG or JPEG image can go in the report.", http.StatusUnprocessableEntity)
		return
	}
	media := map[string]string{"png": "image/png", "jpeg": "image/jpeg"}[format]
	if media == "" {
		http.Error(w, "Only a PNG or JPEG image can go in the report.", http.StatusUnprocessableEntity)
		return
	}
	blob, err := deps.Blobs.Put(r.Context(), bytes.NewReader(data))
	if err != nil {
		http.Error(w, "the image could not be stored", http.StatusInternalServerError)
		return
	}
	if err := deps.Store.SaveReportImage(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, blob.SHA256, blob.Size, media); err != nil {
		if errors.Is(err, store.ErrReportImage) {
			http.Error(w, "Only a PNG or JPEG image can go in the report.", http.StatusUnprocessableEntity)
			return
		}
		reportError(w, err)
		return
	}
	sum := hex.EncodeToString(blob.SHA256)
	writeJSON(w, http.StatusCreated, imageReply{SHA256: sum, Line: imageLine(r.URL.Query().Get("caption"), sum)})
}

// getReportImage serves an image of this report, inline: it was decoded
// as an image when it was saved.
func getReportImage(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, false)
	if !ok {
		return
	}
	sum, media, err := deps.Store.ReportImage(r.Context(), session.OrgID, r.PathValue("r"), r.PathValue("sha"))
	if err != nil {
		reportError(w, err)
		return
	}
	f, err := deps.Blobs.Open(sum)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", media)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	_, _ = io.Copy(w, f)
}

// placeReportImage puts a filed image document of the project in a cell:
// the image is listed against the report, and its line is added under the
// cell's words.
func placeReportImage(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := reportSession(w, r, deps, true)
	if !ok {
		return
	}
	var body struct {
		store.RowWrite
		Document string `json:"document"`
		Column   string `json:"column"`
		Caption  string `json:"caption"`
		Version  int64  `json:"version"`
	}
	if err := readProposalJSON(w, r, deps.MaxBodyBytes, &body); err != nil || body.Document == "" || body.Column == "" {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	sum, name, err := deps.Store.PlaceReportImage(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, body.Document)
	if err != nil {
		if errors.Is(err, store.ErrReportImage) {
			http.Error(w, "That document is not a PNG or JPEG image.", http.StatusUnprocessableEntity)
			return
		}
		reportError(w, err)
		return
	}
	caption := body.Caption
	if caption == "" {
		caption = name
	}
	view, err := deps.Store.ReadReport(r.Context(), session.OrgID, r.PathValue("r"), deps.ReportBuild)
	if err != nil || view.Draft == nil {
		reportError(w, store.ErrNotFound)
		return
	}
	t, _ := findTable(view.Draft, body.Table)
	if t == nil {
		http.NotFound(w, r)
		return
	}
	current := ""
	for i, f := range t.Meta {
		if body.Row != "" && f.ID == body.Row || body.Row == "" && f.Key == body.Key {
			if col, err := columnOf(t, body.Column); err == nil {
				current = t.Rows[i][col]
				body.Column = fmt.Sprint(col)
			}
		}
	}
	text := strings.TrimSpace(strings.TrimSpace(current) + "\n" + imageLine(caption, sum))
	body.Cells = map[string]*string{body.Column: &text}
	body.Source = body.Document
	draft, id, err := deps.Store.WriteReportRow(r.Context(), session.OrgID, r.PathValue("r"), session.UserID, body.Version, body.RowWrite)
	if err != nil {
		reportRowError(w, err)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(session.OrgID)
	}
	writeJSON(w, http.StatusOK, rowReply{Draft: draft, Row: id})
}

// placeImageOperation puts a filed image in a cell, from chat.
func placeImageOperation() operation {
	return forward{Name: "reports.place_image", Undo: true,
		About:  "Put a filed image document (a PNG or JPEG of this project) in a cell of the project report's tables, under its words.",
		Fields: `table (a table id from read report); row (its key or its first cell's words); column (its heading or index); document (the image document's id); caption (words under it; leave out for its file name)`,
		Route:  "PUT /reports/{r}/images/place",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			r, dr, err := reportDraftFor(ctx, d, who, project)
			if err != nil {
				return sent{}, err
			}
			t, ids := findTable(dr, fieldText(f, "table"))
			if t == nil {
				return sent{}, refuse("table", "The report's tables are %s.", strings.Join(ids, ", "))
			}
			i, err := rowOf(t, fieldText(f, "row"))
			if err != nil {
				return sent{}, err
			}
			body := rowTarget(t, i)
			body["column"], body["document"], body["caption"], body["version"] = fieldText(f, "column"), fieldText(f, "document"), fieldText(f, "caption"), dr.Version
			return sent{Path: map[string]string{"r": r.ID}, Body: body}, nil
		},
		Done: func(f map[string]any) string {
			return "Put the image in " + fieldText(f, "table") + ", " + fieldText(f, "row") + "."
		},
	}.operation()
}
