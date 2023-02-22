package errorhandling

import (
	"fmt"
	"strings"

	"github.com/teamkeel/keel/colors"
	"github.com/teamkeel/keel/schema/reader"
)

// ToAnnotatedSchema formats the validation errors by pointing to the relevant line
// in the source file that produced the error
//
// The output is formatted using ANSI colours (if supported by the environment).
func (verrs *ValidationErrors) ToAnnotatedSchema(sources []reader.SchemaFile) string {

	// Number of lines of the source code to render before and after the line with the error
	bufferLines := 3

	// How much to indent the entire result e.g. every line is indented this much
	indent := 2

	result := strings.Repeat(" ", indent)
	newLine := func() {
		result += "\n" + strings.Repeat(" ", indent)
	}

	for _, err := range verrs.Errors {
		// Assumption here is that the error is on one line
		errorLine := err.Pos.Line

		startSourceLine := errorLine - bufferLines
		endSourceLine := errorLine + bufferLines

		// This produces a format string like "%3s| " which we use to render the gutter
		// The number after the "%" is the width, which is documented as:
		//   > For most values, width is the minimum number of runes to output,
		//   > padding the formatted form with spaces if necessary.
		gutterFmt := "%" + fmt.Sprintf("%d", len(fmt.Sprintf("%d", endSourceLine))) + "s| "

		var source string
		for _, s := range sources {
			if s.FileName == err.Pos.Filename {
				source = s.Contents
				break
			}
		}

		result += colors.Gray(fmt.Sprintf(gutterFmt, " ")).String()
		result += colors.Green(fmt.Sprint(err.Pos.Filename)).String()
		newLine()

		// not sure this can happen, but just in case we'll handle it
		if source == "" {
			result += err.Message
			newLine()
			continue
		}

		lines := strings.Split(source, "\n")

		for lineIndex, line := range lines {

			// If this line is outside of the buffer we can drop it
			if (lineIndex+1) < (startSourceLine) || (lineIndex+1) > (endSourceLine) {
				continue
			}

			// Render line numbers in gutter
			result += colors.Gray(fmt.Sprintf(gutterFmt, fmt.Sprintf("%d", lineIndex+1))).String()

			// If the error line doesn't match the currently enumerated line
			// then we can render the whole line without any colorization
			if (lineIndex + 1) != errorLine {
				result += colors.Gray(line).String()
				newLine()
				continue
			}

			chars := strings.Split(line, "")

			// Enumerate over the characters in the line
			for charIdx, char := range chars {

				// Check if the character index is less than or greater than the corresponding start and end column
				// If so, then render the char without any colorization
				if (charIdx+1) < err.Pos.Column || (charIdx+1) > err.EndPos.Column-1 {
					result += char
					continue
				}

				result += colors.Red(fmt.Sprint(char)).String()
			}

			newLine()

			// Underline the token that caused the error
			result += colors.Gray(fmt.Sprintf(gutterFmt, "")).String()
			result += strings.Repeat(" ", err.Pos.Column-1)
			tokenLength := err.EndPos.Column - err.Pos.Column
			for i := 0; i < tokenLength; i++ {
				if i == tokenLength/2 {
					result += colors.Yellow("\u252C").Highlight().String()
				} else {
					result += colors.Yellow("\u2500").Highlight().String()
				}
			}
			newLine()

			// Render the down arrow
			result += colors.Gray(fmt.Sprintf(gutterFmt, "")).String()
			result += strings.Repeat(" ", err.Pos.Column-1)
			result += strings.Repeat(" ", (err.EndPos.Column-err.Pos.Column)/2)
			result += colors.Yellow("\u2570").Highlight().String()
			result += colors.Yellow("\u2500").Highlight().String()

			// Render the message
			result += fmt.Sprintf(" %s %s", colors.Yellow(fmt.Sprint(err.ErrorDetails.Message)).Highlight().String(), colors.Red(fmt.Sprintf("(%s)", err.Code)).String())
			newLine()

			// Render the hint
			if err.ErrorDetails.Hint != "" {
				result += colors.Gray(fmt.Sprintf(gutterFmt, "")).String()
				result += strings.Repeat(" ", err.Pos.Column-1)
				result += strings.Repeat(" ", (err.EndPos.Column-err.Pos.Column)/2)
				// Line up hint with the error message above (taking into account unicode arrows)
				result += strings.Repeat(" ", 3)
				result += colors.Cyan(fmt.Sprint(err.ErrorDetails.Hint)).String()
				newLine()
			}
		}

		newLine()
	}

	return result
}
