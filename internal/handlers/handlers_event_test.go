package handlers

import (
	"testing"

	"futsal-bot/internal/models"
)

func TestSessionDelta(t *testing.T) {
	present := &models.EventResponse{Response: models.ResponsePresent}
	absent := &models.EventResponse{Response: models.ResponseAbsent}

	tests := []struct {
		name     string
		previous *models.EventResponse
		response models.EventResponseType
		want     int
	}{
		{"first answer present charges one session", nil, models.ResponsePresent, 1},
		{"first answer absent charges nothing", nil, models.ResponseAbsent, 0},
		{"present again is not charged twice", present, models.ResponsePresent, 0},
		{"absent again changes nothing", absent, models.ResponseAbsent, 0},
		{"cancelling gives the session back", present, models.ResponseAbsent, -1},
		{"joining after cancelling charges again", absent, models.ResponsePresent, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionDelta(tt.previous, tt.response); got != tt.want {
				t.Errorf("sessionDelta() = %d, want %d", got, tt.want)
			}
		})
	}
}

// Tapping the same button many times, or flip-flopping, must never drift the
// member's balance away from what their final answer implies.
func TestSessionDeltaDoesNotDrift(t *testing.T) {
	taps := []models.EventResponseType{
		models.ResponsePresent,
		models.ResponsePresent,
		models.ResponseAbsent,
		models.ResponseAbsent,
		models.ResponsePresent,
		models.ResponseAbsent,
		models.ResponsePresent,
	}

	var previous *models.EventResponse
	balance := 0

	for _, tap := range taps {
		balance += sessionDelta(previous, tap)
		previous = &models.EventResponse{Response: tap}
	}

	if balance != 1 {
		t.Errorf("balance after ending on present = %d, want 1", balance)
	}
}

func TestBuildEventBoard(t *testing.T) {
	event := &models.Event{Month: "مرداد", SessionDate: "جمعه ۱۵", Capacity: 14}
	answers := []models.EventAnswer{
		{Name: "علی", Response: models.ResponsePresent},
		{Name: "حسین", Response: models.ResponseAbsent},
		{Name: "سارا", Response: models.ResponsePresent},
	}

	want := `بسم الله

 *مرداد ماه*
حاضرین و غایبین سالن *جمعه ۱۵* : فوتسال

🔹 لطفا تا قبل ساعت ۱۲ جمعه اعلام حضور کنین.
🔹 ️لطفا به موقع تشریف بیاورید.

                      ⭕️ *ظرفیت 14 نفر* ⭕️

🔹 حاضرین :
1- علی
2- سارا
🔻 غایبین:
1- حسین`

	if got := buildEventBoard(event, answers, nil); got != want {
		t.Errorf("buildEventBoard() mismatch\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

// Guests are appended to the present list, keep the numbering running, and are
// labelled so the group can tell them apart from members.
func TestBuildEventBoardWithGuests(t *testing.T) {
	event := &models.Event{Month: "مرداد", SessionDate: "جمعه ۱۵", Capacity: 14}
	answers := []models.EventAnswer{
		{Name: "علی", Response: models.ResponsePresent},
		{Name: "حسین", Response: models.ResponseAbsent},
		{Name: "سارا", Response: models.ResponsePresent},
	}
	guests := []models.EventGuest{
		{Name: "رضا"},
		{Name: "محمد"},
	}

	want := `🔹 حاضرین :
1- علی
2- سارا
3- رضا (مهمان)
4- محمد (مهمان)
🔻 غایبین:
1- حسین`

	if got := buildEventBoard(event, answers, guests); !contains(got, want) {
		t.Errorf("guest lines wrong\ngot:\n%s\n\nwant to contain:\n%s", got, want)
	}
}

// A session with only guests and nobody else must still number from 1.
func TestBuildEventBoardGuestsOnly(t *testing.T) {
	event := &models.Event{Month: "مرداد", SessionDate: "جمعه ۱۵", Capacity: 14}
	guests := []models.EventGuest{{Name: "رضا"}}

	want := `🔹 حاضرین :
1- رضا (مهمان)
🔻 غایبین:
`

	if got := buildEventBoard(event, nil, guests); !contains(got, want) {
		t.Errorf("guests-only board wrong\ngot:\n%s\n\nwant to contain:\n%s", got, want)
	}
}

// Guest names are typed by an admin, so they need the same Markdown escaping as
// member names or the whole board stops rendering.
func TestBuildEventBoardEscapesGuestNames(t *testing.T) {
	event := &models.Event{Month: "مرداد", SessionDate: "جمعه ۱۵", Capacity: 14}
	guests := []models.EventGuest{{Name: "reza_*x*"}}

	got := buildEventBoard(event, nil, guests)
	if want := "1- reza\\_\\*x\\* (مهمان)"; !contains(got, want) {
		t.Errorf("guest name was not escaped, want %q in:\n%s", want, got)
	}
}

func TestBuildEventBoardEmptyLists(t *testing.T) {
	event := &models.Event{Month: "مرداد", SessionDate: "جمعه ۱۵", Capacity: 14}

	want := `بسم الله

 *مرداد ماه*
حاضرین و غایبین سالن *جمعه ۱۵* : فوتسال

🔹 لطفا تا قبل ساعت ۱۲ جمعه اعلام حضور کنین.
🔹 ️لطفا به موقع تشریف بیاورید.

                      ⭕️ *ظرفیت 14 نفر* ⭕️

🔹 حاضرین :

🔻 غایبین:
`

	if got := buildEventBoard(event, nil, nil); got != want {
		t.Errorf("buildEventBoard() with no answers mismatch\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestBuildEventBoardClosed(t *testing.T) {
	event := &models.Event{Month: "مرداد", SessionDate: "جمعه ۱۵", Capacity: 14, IsClosed: true}

	got := buildEventBoard(event, nil, nil)
	if want := "🔒 اعلام حضور بسته شد."; got[len(got)-len(want):] != want {
		t.Errorf("closed board should end with the closed notice, got:\n%s", got)
	}
}

// A name containing Markdown characters must not be able to break the message,
// otherwise the whole board fails to send and stops updating.
func TestBuildEventBoardEscapesNames(t *testing.T) {
	event := &models.Event{Month: "مرداد", SessionDate: "جمعه ۱۵", Capacity: 14}
	answers := []models.EventAnswer{
		{Name: "ali_*star*", Response: models.ResponsePresent},
	}

	got := buildEventBoard(event, answers, nil)
	if want := "1- ali\\_\\*star\\*"; !contains(got, want) {
		t.Errorf("name was not escaped, want %q in:\n%s", want, got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
