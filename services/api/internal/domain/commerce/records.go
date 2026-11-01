package commerce

import "github.com/google/uuid"

// indexRecords maps option ids to their purchase state.
func indexRecords(records []OptionRecord) map[uuid.UUID]OptionRecord {
	out := make(map[uuid.UUID]OptionRecord, len(records))
	for _, record := range records {
		out[record.ID] = record
	}
	return out
}
