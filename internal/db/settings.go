package db

func (d *DB) GetSettings() (Settings, error) {
	var s Settings
	err := d.SQL.QueryRow(`SELECT font_size, line_height, theme, accent FROM reader_settings WHERE id = 1`).Scan(&s.FontSize, &s.LineHeight, &s.Theme, &s.Accent)
	if err != nil {
		return Settings{FontSize: 18, LineHeight: 1.6, Theme: "light", Accent: "emerald"}, err
	}
	s.Accent = normalizeAccent(s.Accent)
	return s, nil
}

func (d *DB) UpdateSettings(s Settings) error {
	if s.FontSize < 12 {
		s.FontSize = 12
	}
	if s.FontSize > 36 {
		s.FontSize = 36
	}
	if s.LineHeight < 1.2 {
		s.LineHeight = 1.2
	}
	if s.LineHeight > 2.4 {
		s.LineHeight = 2.4
	}
	switch s.Theme {
	case "light", "dark", "sepia":
	default:
		s.Theme = "light"
	}
	if s.Accent == "" {
		if cur, err := d.GetSettings(); err == nil {
			s.Accent = cur.Accent
		}
	}
	s.Accent = normalizeAccent(s.Accent)
	_, err := d.SQL.Exec(`UPDATE reader_settings SET font_size=?, line_height=?, theme=?, accent=? WHERE id=1`, s.FontSize, s.LineHeight, s.Theme, s.Accent)
	return err
}

func normalizeAccent(s string) string {
	switch s {
	case "amber", "orange", "rose", "red", "emerald", "teal", "sky", "indigo", "violet", "pink":
		return s
	default:
		return "emerald"
	}
}
