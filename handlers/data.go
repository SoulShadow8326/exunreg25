package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"exunreg25/db"

	"google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"
)

func parseFlexibleTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02 15:04:05 -0700 MST",
	}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, s, time.UTC); err == nil {
			return t, nil
		}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized time format: %s", s)
}

var (
	sheetsResetCh chan struct{}
	sheetsOpMu    sync.Mutex
)

func startSheetsSync(database *db.Database) {
	if sheetsResetCh == nil {
		sheetsResetCh = make(chan struct{}, 1)
	}
	interval := 1 * time.Minute
	timer := time.NewTimer(interval)
	defer timer.Stop()

	backupInterval := 360 * time.Minute
	if v := os.Getenv("DRIVE_BACKUP_INTERVAL"); v != "" {
		if m, err := strconv.Atoi(v); err == nil && m > 0 {
			backupInterval = time.Duration(m) * time.Minute
		}
	}
	backupTicker := time.NewTicker(backupInterval)
	defer backupTicker.Stop()

	if err := syncAllTablesToSheets(database); err != nil {
		log.Printf("sheets sync initial run error: %v", err)
	}

	for {
		select {
		case <-timer.C:
			sheetsOpMu.Lock()
			log.Printf("starting sheets sync")
			if err := syncAllTablesToSheets(database); err != nil {
				log.Printf("sheets sync error: %v", err)
			} else {
				log.Printf("sheets sync completed")
			}
			sheetsOpMu.Unlock()
			timer.Reset(interval)
		case <-sheetsResetCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			sheetsOpMu.Lock()
			log.Printf("starting sheets sync (manual trigger)")
			if err := syncAllTablesToSheets(database); err != nil {
				log.Printf("sheets sync error: %v", err)
			} else {
				log.Printf("sheets sync completed (manual trigger)")
			}
			sheetsOpMu.Unlock()
			timer.Reset(interval)
		case <-backupTicker.C:
			sheetsOpMu.Lock()
			saJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")
			if saJSON != "" {
				if _, err := os.Stat(saJSON); err == nil {
					b, err := os.ReadFile(saJSON)
					if err == nil {
						dbPath := os.Getenv("DB_PATH")
						if dbPath == "" {
							dbPath = "./data/exunreg25.db"
						}
						log.Printf("scheduled Drive backup for %s", dbPath)
						id, err := UploadDBBackupToDrive(b, dbPath)
						if err != nil {
							log.Printf("drive backup error: %v", err)
						} else {
							log.Printf("drive backup uploaded id: %s", id)
						}
					}
				}
			}
			sheetsOpMu.Unlock()
		}
	}
}

func syncAllTablesToSheets(database *db.Database) error {
	saJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")
	if saJSON == "" {
		return fmt.Errorf("GOOGLE_SERVICE_ACCOUNT_JSON not set")
	}
	if _, err := os.Stat(saJSON); err == nil {
		b, err := os.ReadFile(saJSON)
		if err != nil {
			return fmt.Errorf("failed to read service account file: %v", err)
		}
		saJSON = string(b)
	}
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	if spreadsheetID == "" {
		return fmt.Errorf("SPREADSHEET_ID not set")
	}

	ctx := context.Background()
	creds := []byte(saJSON)
	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(creds))
	if err != nil {
		return fmt.Errorf("failed to create sheets service: %v", err)
	}

	tables, err := listTables(database)
	if err != nil {
		return fmt.Errorf("failed to list tables: %v", err)
	}

	primaryKeys := map[string]string{
		"users":                    "email",
		"events":                   "id",
		"individual_registrations": "id",
	}

	for _, t := range tables {
		rows, err := queryTableRows(database, t)
		if err != nil {
			log.Printf("failed to query table %s: %v", t, err)
			continue
		}

		sheetName := t
		sheetId, err := ensureSheetExists(ctx, srv, spreadsheetID, sheetName)
		if err != nil {
			log.Printf("failed to ensure sheet %s exists: %v", sheetName, err)
			continue
		}

		hdr, sheetRows, err := readSheetRows(ctx, srv, spreadsheetID, sheetName)
		if err != nil {
			log.Printf("failed to read sheet %s: %v", sheetName, err)
			vr := &sheets.ValueRange{Values: convertToValues(rows)}
			rangeA1 := sheetName + "!A1"
			_, err = srv.Spreadsheets.Values.Update(spreadsheetID, rangeA1, vr).ValueInputOption("RAW").Do()
			if err != nil {
				log.Printf("failed to update sheet %s: %v", sheetName, err)
			}
			continue
		}

		pk, ok := primaryKeys[t]
		if ok {
			pkIndex := -1
			idIndex := -1
			for i, h := range hdr {
				if strings.EqualFold(h, pk) {
					pkIndex = i
				}
				if strings.EqualFold(h, "id") {
					idIndex = i
				}
			}
			sheetPKs := make(map[string]bool)
			if pkIndex >= 0 || idIndex >= 0 {
				seenPK := map[string]int{}
				seenID := map[string]int{}
				toDeleteSet := map[int]bool{}
				dbDeleteIDs := []string{}
				for i, r := range sheetRows {
					var pkVal string
					if pkIndex >= 0 && pkIndex < len(r) {
						pkVal = strings.TrimSpace(r[pkIndex])
						if pkVal == "''" {
							pkVal = ""
						}
					}
					var idVal string
					if idIndex >= 0 && idIndex < len(r) {
						idVal = strings.TrimSpace(r[idIndex])
						if idVal == "''" {
							idVal = ""
						}
					}
					if pkVal != "" {
						if first, ok := seenPK[pkVal]; ok {
							toDeleteSet[i] = true
							_ = first
							continue
						}
						seenPK[pkVal] = i
					}
					if idVal != "" {
						if first, ok := seenID[idVal]; ok {
							toDeleteSet[i] = true
							_ = first
							continue
						}
						seenID[idVal] = i
					}
					if idVal != "" && pkVal == "" {
						allEmpty := true
						for j, cell := range r {
							if j == idIndex {
								continue
							}
							c := strings.TrimSpace(cell)
							if c == "''" {
								c = ""
							}
							if c != "" {
								allEmpty = false
								break
							}
						}
						if allEmpty {
							toDeleteSet[i] = true
							dbDeleteIDs = append(dbDeleteIDs, idVal)
						}
					}
				}
				if len(toDeleteSet) > 0 {
					idxs := make([]int, 0, len(toDeleteSet))
					for k := range toDeleteSet {
						idxs = append(idxs, k)
					}
					sort.Slice(idxs, func(a, b int) bool { return idxs[a] > idxs[b] })
					requests := []*sheets.Request{}
					for _, sr := range idxs {
						startIndex := int64(sr + 1)
						endIndex := startIndex + 1
						deleteReq := &sheets.DeleteDimensionRequest{
							Range: &sheets.DimensionRange{
								SheetId:    sheetId,
								Dimension:  "ROWS",
								StartIndex: startIndex,
								EndIndex:   endIndex,
							},
						}
						requests = append(requests, &sheets.Request{DeleteDimension: deleteReq})
					}
					if len(requests) > 0 {
						batch := &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}
						_, _ = srv.Spreadsheets.BatchUpdate(spreadsheetID, batch).Context(ctx).Do()
					}
					if len(dbDeleteIDs) > 0 {
						for _, did := range dbDeleteIDs {
							dq := fmt.Sprintf("DELETE FROM %s WHERE id = ?", t)
							_, _ = database.Exec(dq, did)
						}
					}
					newRows := make([][]string, 0, len(sheetRows)-len(toDeleteSet))
					for i, r := range sheetRows {
						if _, del := toDeleteSet[i]; del {
							continue
						}
						newRows = append(newRows, r)
					}
					sheetRows = newRows
				}
				for _, r := range sheetRows {
					if pkIndex >= 0 && pkIndex < len(r) {
						v := strings.TrimSpace(r[pkIndex])
						if v == "''" {
							v = ""
						}
						if v != "" {
							sheetPKs[v] = true
						}
					}
				}
			}
			if len(sheetPKs) > 0 {
				if err := deleteDBRowsNotInSheet(database, t, pk, sheetPKs); err != nil {
					log.Printf("failed to delete missing rows for table %s: %v", t, err)
				}
			} else {
				log.Printf("sheet %s has no primary-key values; skipping delete to avoid wiping DB", sheetName)
			}
			if err := applyTableUpserts(database, t, pk, hdr, sheetRows); err != nil {
				log.Printf("failed to apply upserts for table %s: %v", t, err)
			}
			rows, err = queryTableRows(database, t)
			if err != nil {
				log.Printf("failed to re-query table %s: %v", t, err)
				continue
			}
		}

		pkForUpdate := ""
		if p, ok := primaryKeys[t]; ok {
			pkForUpdate = p
		}
		if err := updateOnlyMissingCells(ctx, srv, spreadsheetID, sheetName, sheetId, pkForUpdate, hdr, sheetRows, rows); err != nil {
			log.Printf("failed to partially update sheet %s: %v", sheetName, err)
			continue
		}

		if len(rows) > 0 {
			colsCount := len(rows[0])
			requests := []*sheets.Request{}
			updateGrid := &sheets.UpdateSheetPropertiesRequest{
				Properties: &sheets.SheetProperties{
					SheetId: sheetId,
					GridProperties: &sheets.GridProperties{
						FrozenRowCount: 1,
					},
				},
				Fields: "gridProperties.frozenRowCount",
			}
			requests = append(requests, &sheets.Request{UpdateSheetProperties: updateGrid})

			repeat := &sheets.Request{RepeatCell: &sheets.RepeatCellRequest{
				Range: &sheets.GridRange{
					SheetId:          sheetId,
					StartRowIndex:    0,
					EndRowIndex:      1,
					StartColumnIndex: 0,
					EndColumnIndex:   int64(colsCount),
				},
				Cell:   &sheets.CellData{UserEnteredFormat: &sheets.CellFormat{TextFormat: &sheets.TextFormat{Bold: true}}},
				Fields: "userEnteredFormat.textFormat.bold",
			}}
			requests = append(requests, repeat)
			batch := &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}
			_, _ = srv.Spreadsheets.BatchUpdate(spreadsheetID, batch).Context(ctx).Do()
		}
	}

	return nil
}

func listTables(database *db.Database) ([]string, error) {
	rows, err := database.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func queryTableRows(database *db.Database, table string) ([][]interface{}, error) {
	q := fmt.Sprintf("SELECT * FROM %s", table)
	rows, err := database.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := [][]interface{}{}
	header := make([]interface{}, len(cols))
	for i, c := range cols {
		header[i] = c
	}
	result = append(result, header)

	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		rec := make([]interface{}, len(cols))
		for i, v := range vals {
			switch val := v.(type) {
			case nil:
				rec[i] = ""
			case []byte:
				rec[i] = string(val)
			default:
				rec[i] = fmt.Sprintf("%v", val)
			}
		}
		result = append(result, rec)
	}
	return result, nil
}

func convertToValues(rows [][]interface{}) [][]interface{} {
	vals := make([][]interface{}, len(rows))
	for i, r := range rows {
		row := make([]interface{}, len(r))
		copy(row, r)
		vals[i] = row
	}
	return vals
}

func ensureSheetExists(ctx context.Context, srv *sheets.Service, spreadsheetID, sheetName string) (int64, error) {
	ss, err := srv.Spreadsheets.Get(spreadsheetID).Fields("sheets.properties").Do()
	if err != nil {
		return 0, err
	}
	for _, s := range ss.Sheets {
		if s.Properties.Title == sheetName {
			return s.Properties.SheetId, nil
		}
	}
	addReq := &sheets.Request{AddSheet: &sheets.AddSheetRequest{Properties: &sheets.SheetProperties{Title: sheetName}}}
	batch := &sheets.BatchUpdateSpreadsheetRequest{Requests: []*sheets.Request{addReq}}
	resp, err := srv.Spreadsheets.BatchUpdate(spreadsheetID, batch).Context(ctx).Do()
	if err != nil {
		return 0, err
	}
	if len(resp.Replies) > 0 && resp.Replies[0].AddSheet != nil && resp.Replies[0].AddSheet.Properties != nil {
		return resp.Replies[0].AddSheet.Properties.SheetId, nil
	}
	ss2, err := srv.Spreadsheets.Get(spreadsheetID).Fields("sheets.properties").Do()
	if err != nil {
		return 0, err
	}
	for _, s := range ss2.Sheets {
		if s.Properties.Title == sheetName {
			return s.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("failed to get sheet id for %s", sheetName)
}

func readSheetRows(ctx context.Context, srv *sheets.Service, spreadsheetID, sheetName string) ([]string, [][]string, error) {
	rng := sheetName + "!A1:Z"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, rng).Context(ctx).Do()
	if err != nil {
		return nil, nil, err
	}
	if len(resp.Values) == 0 {
		return nil, nil, fmt.Errorf("empty sheet")
	}
	headerIface := resp.Values[0]
	headers := make([]string, len(headerIface))
	for i, h := range headerIface {
		headers[i] = fmt.Sprintf("%v", h)
	}
	rows := [][]string{}
	for _, r := range resp.Values[1:] {
		row := make([]string, len(headers))
		for i := range headers {
			if i < len(r) {
				row[i] = fmt.Sprintf("%v", r[i])
			} else {
				row[i] = ""
			}
		}
		rows = append(rows, row)
	}
	return headers, rows, nil
}

func applyTableUpserts(database *db.Database, table, pk string, headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return fmt.Errorf("no headers")
	}
	updatedAtIndex := -1
	for i, h := range headers {
		if strings.EqualFold(h, "updated_at") {
			updatedAtIndex = i
			break
		}
	}
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	for _, r := range rows {
		values := map[string]string{}
		for i, h := range headers {
			v := r[i]
			if strings.TrimSpace(v) == "''" {
				v = ""
			}
			values[h] = v
		}
		keyVal := values[pk]
		if keyVal == "" {
			continue
		}
		skipDueToTimestamp := false
		if updatedAtIndex != -1 {
			q := fmt.Sprintf("SELECT updated_at FROM %s WHERE %s = ?", table, pk)
			var dbUpdated sql.NullString
			err := database.QueryRow(q, keyVal).Scan(&dbUpdated)
			if err == nil {
				sheetUpdatedStr := strings.TrimSpace(values[headers[updatedAtIndex]])
				if sheetUpdatedStr == "''" {
					sheetUpdatedStr = ""
				}
				sheetUpdated, _ := parseFlexibleTime(sheetUpdatedStr)
				dbUpdatedTime, _ := parseFlexibleTime(dbUpdated.String)
				if sheetUpdated.IsZero() || !sheetUpdated.After(dbUpdatedTime) {
					skipDueToTimestamp = true
				}
			}
		}
		if skipDueToTimestamp {
			qCols := make([]string, len(headers))
			copy(qCols, headers)
			q := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", strings.Join(qCols, ", "), table, pk)
			row := database.QueryRow(q, keyVal)
			dbVals := make([]sql.NullString, len(headers))
			ptrs := make([]interface{}, len(headers))
			for i := range dbVals {
				ptrs[i] = &dbVals[i]
			}
			err := row.Scan(ptrs...)
			allowBecauseDiff := false
			if err == nil {
				for i, h := range headers {
					if strings.EqualFold(h, pk) {
						continue
					}
					sheetVal := strings.TrimSpace(values[h])
					if sheetVal == "''" {
						sheetVal = ""
					}
					if sheetVal == "" {
						continue
					}
					dbVal := ""
					if dbVals[i].Valid {
						dbVal = dbVals[i].String
					}
					if sheetVal != dbVal {
						allowBecauseDiff = true
						break
					}
				}
			} else {
				allowBecauseDiff = true
			}
			if !allowBecauseDiff {
				log.Printf("skipping upsert for table %s key %s because sheet is not newer than DB", table, keyVal)
				continue
			}
			log.Printf("applying upsert for table %s key %s because sheet has differing values", table, keyVal)
		}
		cols := []string{}
		placeholders := []string{}
		args := []interface{}{}
		updates := []string{}
		for _, h := range headers {
			cols = append(cols, h)
			placeholders = append(placeholders, "?")
			v := values[h]
			vtrim := strings.TrimSpace(v)
			if vtrim == "" {
				args = append(args, nil)
			} else {
				args = append(args, v)
			}
			if h != pk {
				updates = append(updates, fmt.Sprintf("%s = COALESCE(excluded.%s, %s)", h, h, h))
			}
		}
		q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) DO UPDATE SET %s", table, strings.Join(cols, ", "), strings.Join(placeholders, ", "), pk, strings.Join(updates, ", "))
		if _, err := tx.Exec(q, args...); err != nil {
			log.Printf("upsert error table %s key %s: %v", table, keyVal, err)
			continue
		}
	}
	return nil
}

func deleteDBRowsNotInSheet(database *db.Database, table, pk string, sheetPKs map[string]bool) error {
	q := fmt.Sprintf("SELECT %s FROM %s", pk, table)
	rows, err := database.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()
	var toDelete []string
	for rows.Next() {
		var val sql.NullString
		if err := rows.Scan(&val); err != nil {
			return err
		}
		if !val.Valid {
			continue
		}
		s := val.String
		if _, ok := sheetPKs[s]; !ok {
			toDelete = append(toDelete, s)
		}
	}
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	for _, d := range toDelete {
		dq := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table, pk)
		if _, err := tx.Exec(dq, d); err != nil {
			log.Printf("failed to delete %s=%s: %v", pk, d, err)
			continue
		}
	}
	return nil
}

func TriggerSheetsSync() error {
	if globalDB == nil {
		return fmt.Errorf("database not initialized")
	}
	if sheetsResetCh == nil {
		sheetsResetCh = make(chan struct{}, 1)
	}
	select {
	case sheetsResetCh <- struct{}{}:
	default:
	}
	return nil
}

func a1ColumnName(n int) string {
	name := ""
	for n >= 0 {
		ch := rune('A' + (n % 26))
		name = string([]rune{ch}) + name
		n = n/26 - 1
	}
	return name
}

func updateOnlyMissingCells(ctx context.Context, srv *sheets.Service, spreadsheetID, sheetName string, sheetId int64, pk string, headers []string, sheetRows [][]string, dbRows [][]interface{}) error {
	headerCount := len(headers)
	sheetMap := map[string]int{}
	pkIndex := -1
	if headerCount == 0 {
		return fmt.Errorf("no headers")
	}
	if pk == "" {
		pk = headers[0]
	}
	for i, h := range headers {
		if strings.EqualFold(h, pk) {
			pkIndex = i
			break
		}
	}
	if pkIndex == -1 {
		pkIndex = 0
	}
	for i, r := range sheetRows {
		if pkIndex < len(r) {
			v := strings.TrimSpace(r[pkIndex])
			if v == "''" {
				v = ""
			}
			sheetMap[v] = i
		}
	}

	valueRanges := []*sheets.ValueRange{}
	appendRows := [][]interface{}{}

	for ri, dbRow := range dbRows[1:] {
		pkVal := fmt.Sprintf("%v", dbRow[pkIndex])
		if pkVal == "" {
			continue
		}
		if sr, exists := sheetMap[pkVal]; exists {
			for ci := 0; ci < headerCount; ci++ {
				var sheetVal string
				if ci < len(sheetRows[sr]) {
					sheetVal = sheetRows[sr][ci]
				}
				dbVal := ""
				if ci < len(dbRow) {
					dbVal = fmt.Sprintf("%v", dbRow[ci])
				}
				writeVal := dbVal
				if dbVal == "" {
					writeVal = ""
				}
				if sheetVal == "" {
					col := a1ColumnName(ci)
					rowNum := sr + 2
					rng := fmt.Sprintf("%s!%s%d", sheetName, col, rowNum)
					vr := &sheets.ValueRange{Range: rng, Values: [][]interface{}{{writeVal}}}
					valueRanges = append(valueRanges, vr)
				}
			}
		} else {
			newRow := make([]interface{}, headerCount)
			for ci := 0; ci < headerCount; ci++ {
				if ci < len(dbRow) {
					dv := fmt.Sprintf("%v", dbRow[ci])
					if dv == "" {
						newRow[ci] = ""
					} else {
						newRow[ci] = dv
					}
				} else {
					newRow[ci] = ""
				}
			}
			appendRows = append(appendRows, newRow)
		}
		_ = ri
	}

	if len(valueRanges) > 0 {
		for _, vr := range valueRanges {
			_, _ = srv.Spreadsheets.Values.Update(spreadsheetID, vr.Range, vr).ValueInputOption("RAW").Context(ctx).Do()
		}
	}
	if len(appendRows) > 0 {
		appendVR := &sheets.ValueRange{Values: appendRows}
		_, _ = srv.Spreadsheets.Values.Append(spreadsheetID, sheetName+"!A1", appendVR).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	}

	dbCount := len(dbRows) - 1
	sheetCount := len(sheetRows)
	if sheetCount > dbCount {
		start := dbCount + 1
		end := sheetCount
		deleteReq := &sheets.DeleteDimensionRequest{
			Range: &sheets.DimensionRange{
				SheetId:    sheetId,
				Dimension:  "ROWS",
				StartIndex: int64(start + 1),
				EndIndex:   int64(end + 1),
			},
		}
		batch := &sheets.BatchUpdateSpreadsheetRequest{Requests: []*sheets.Request{{DeleteDimension: deleteReq}}}
		_, _ = srv.Spreadsheets.BatchUpdate(spreadsheetID, batch).Context(ctx).Do()
	}
	return nil
}
