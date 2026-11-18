package delivery

import "sitewise/internal/program"

// DefaultResponseDays is the RFI response time until the PM sets the
// project's own (plan 4, owner decision 2): five working days.
const DefaultResponseDays = 5

// DueDate is an RFI's response date: days working days after it was
// raised, on the Program's calendar (Monday to Friday when there is no
// program). Nil when the raised date is unknown: Due stays blank rather
// than guessed.
func DueDate(cal program.Calendar, raisedOn *string, days int) *string {
	if raisedOn == nil || days < 1 {
		return nil
	}
	due, ok := program.BusinessDayAfter(cal, *raisedOn, days)
	if !ok {
		return nil
	}
	return &due
}
