package treasure

import "time"

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
