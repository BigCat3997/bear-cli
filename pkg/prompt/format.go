package prompt

import (
	"bear_cli/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

func padRight(s string, n int) string {
	return fmt.Sprintf("%-*s", n, s)
}

func PrintTable(data any, hiddenFields ...string) {
	hidden := map[string]bool{}
	for _, h := range hiddenFields {
		hidden[h] = true
	}

	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		s := reflect.MakeSlice(reflect.SliceOf(v.Type()), 1, 1)
		s.Index(0).Set(v)
		v = s
	}

	if v.Len() == 0 {
		// fmt.Println("Empty list")
		return
	}

	var rows []map[string]string
	colNames := map[string]bool{}

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		row := map[string]string{}

		rv := reflect.ValueOf(item)
		rt := reflect.TypeOf(item)

		// Struct support
		if rv.Kind() == reflect.Struct {
			for f := 0; f < rv.NumField(); f++ {
				field := rt.Field(f)
				name := field.Name

				if hidden[name] {
					continue
				}

				val := fmt.Sprintf("%v", rv.Field(f).Interface())
				colNames[name] = true
				row[name] = val
			}
		}

		// Map support
		if rv.Kind() == reflect.Map {
			for _, key := range rv.MapKeys() {
				name := fmt.Sprintf("%v", key.Interface())

				if hidden[name] {
					continue
				}

				val := fmt.Sprintf("%v", rv.MapIndex(key).Interface())
				colNames[name] = true
				row[name] = val
			}
		}

		rows = append(rows, row)
	}

	// Convert colNames to slice
	cols := make([]string, 0, len(colNames))
	for c := range colNames {
		if !hidden[c] {
			cols = append(cols, c)
		}
	}

	// Compute widths
	width := make(map[string]int)
	for _, c := range cols {
		width[c] = len(c)
	}
	for _, row := range rows {
		for c, val := range row {
			if len(val) > width[c] {
				width[c] = len(val)
			}
		}
	}

	// Print header
	for _, c := range cols {
		fmt.Print(padRight(c, width[c]+2))
	}
	fmt.Println()

	for _, c := range cols {
		fmt.Print(strings.Repeat("-", width[c]) + "  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for _, c := range cols {
			fmt.Print(padRight(row[c], width[c]+2))
		}
		fmt.Println()
	}
}

func PrintJSON(body io.Reader) {
	data, err := io.ReadAll(body)
	if err != nil {
		fmt.Println("error reading:", err)
		return
	}

	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "  "); err != nil {
		fmt.Println(string(data)) // not valid JSON → print raw
		return
	}

	fmt.Println(out.String())
}

func PrintJSONText(data any, hiddenFields ...string) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}

func PrintLinuxEnvVar(data any, hiddenFields ...string) error {
	dataMap, ok := data.(map[string]string)
	if !ok {
		return fmt.Errorf("expected map[string]string")
	}
	for k, v := range dataMap {
		fmt.Printf("export %s=%q\n", k, v)
	}
	return nil
}

func PrintStdOut(data any, stdOutFormat models.StdOutFormat, hiddenFields ...string) {
	switch stdOutFormat {
	case models.TABLE:
		PrintTable(data, hiddenFields...)
	case models.JSON:
		PrintJSONText(data, hiddenFields...)
	case models.LINUX_ENV_VAR:
		PrintLinuxEnvVar(data, hiddenFields...)
	default:
		break
	}
}
