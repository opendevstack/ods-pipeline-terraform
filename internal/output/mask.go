package output

import (
	"bytes"
	"io"
)

type MaskWriter struct {
	Writer       io.Writer
	Sensitive    []string
	buffer       bytes.Buffer
	Replacement  rune
	ReplaceValue string
}

func NewMaskWriter(w io.Writer, sensitive []string) *MaskWriter {
	return &MaskWriter{
		Writer:       w,
		Sensitive:    sensitive,
		Replacement:  '*',
		ReplaceValue: "*",
	}
}

func NewMaskWriterFromVars(w io.Writer, sensitiveVars map[string]string) *MaskWriter {
	return NewMaskWriter(w, uniqueValuesToArray(sensitiveVars))
}

func (hw *MaskWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	for _, sensitive := range hw.Sensitive {
		p = bytes.ReplaceAll(p, []byte(sensitive), bytes.Repeat([]byte(string(hw.Replacement)), 3))
	}
	hw.buffer.Write(p)
	_, err = hw.Writer.Write(p)
	return
}

func uniqueValuesToArray(inputMap map[string]string) []string {
	// Map to store unique values as keys and their occurrences
	uniqueValues := make(map[string]bool)
	result := []string{}

	// Iterate through the map
	for _, value := range inputMap {
		// Check if the value is unique
		if _, exists := uniqueValues[value]; !exists {
			uniqueValues[value] = true
			result = append(result, value)
		}
	}
	return result
}
