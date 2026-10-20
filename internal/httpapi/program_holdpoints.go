package httpapi

import (
	"net/http"

	"sitewise/internal/store"
)

// postProgramHoldRelease is POST /program/hold-points/release
// {"version","id","reading_id"}: the user marks one hold point released,
// with a proposed release reading as its evidence or with none
// (reading_id ""). Only this click releases a hold point; a reading is a
// proposal (plan 10.4 "Never automatic"). One undoable batch; no Jev.
func postProgramHoldRelease(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := programSession(w, r, d, true)
	if !ok {
		return
	}
	var body struct {
		Version   int64  `json:"version"`
		ID        string `json:"id"`
		ReadingID string `json:"reading_id"`
	}
	if readProposalJSON(w, r, d.MaxBodyBytes, &body) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Version < 1 {
		programInvalid(w, "version", "The program version is required.")
		return
	}
	if !uuidPattern.MatchString(body.ID) {
		programInvalid(w, "id", store.MsgNotInProgram)
		return
	}
	if body.ReadingID != "" && !uuidPattern.MatchString(body.ReadingID) {
		programInvalid(w, "reading_id", "That evidence is not proposed for this hold point.")
		return
	}
	res, err := d.Store.ReleaseHoldPoint(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, body.Version, body.ID, body.ReadingID)
	if err != nil {
		programError(w, r, d, s.OrgID, res, err)
		return
	}
	writeProgramResult(w, r, d, s.OrgID, res)
}
