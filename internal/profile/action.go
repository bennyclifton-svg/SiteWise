package profile

// ActionApplied uses only this ten-option question's own floor. Never borrow
// presence/provider confidence: https://docs.typesafe.ai/confidence.
func (t Thresholds) ActionApplied(r Reading) bool {
	floor, ok := t.Amber["action"]
	return ok && floor > 0 && r.Confidence != nil && *r.Confidence >= floor && r.Value != "several"
}
