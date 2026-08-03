package calendar

type CalendarFile struct {
	Version   int        `json:"version"`
	Timezone  string     `json:"timezone"`
	Calendars []Calendar `json:"calendars"`
}

type Calendar struct {
	Name       string      `json:"name"`
	Users      []string    `json:"users"`
	Enabled    bool        `json:"enabled"`
	Week       []WeekDay   `json:"week,omitempty"`
	Holidays   []Holiday   `json:"holidays,omitempty"`
	Exceptions []Exception `json:"exceptions,omitempty"`
	Vacations  []Vacation  `json:"vacations,omitempty"`
}

type WeekDay struct {
	WeekDay string  `json:"weekday"`
	DayOff  bool    `json:"day_off"`
	Begin   string  `json:"begin,omitempty"`
	End     string  `json:"end,omitempty"`
	Breaks  []Break `json:"breaks,omitempty"`
}

type Break struct {
	Name  string `json:"name"`
	Begin string `json:"begin"`
	End   string `json:"end"`
}

type Holiday struct {
	Name  string `json:"name"`
	Month int    `json:"month"`
	Day   int    `json:"day"`
}

type Exception struct {
	Date   string `json:"date"`
	DayOff bool   `json:"day_off"`
	Begin  string `json:"begin,omitempty"`
	End    string `json:"end,omitempty"`
}

type Vacation struct {
	From string `json:"from"`
	To   string `json:"to"`
}