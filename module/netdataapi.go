package module

import (
	"fmt"
	"io"
)

type (
	// apiWriter write native netdata orchestrator API
	// https://github.com/firehol/netdata/wiki/External-Plugins#native-netdata-plugin-api
	apiWriter struct {
		// Out write to
		io.Writer
	}
)

// chart defines a new chart.
func (w *apiWriter) chart(
	typeID string,
	ID string,
	name string,
	title string,
	units string,
	family string,
	context string,
	chartType chartType,
	priority int,
	updateEvery int,
	options Opts,
	plugin string,
	module string) error {
	_, err := fmt.Fprintf(w, "CHART '%s.%s' '%s' '%s' '%s' '%s' '%s' '%s' '%d' '%d' '%s' '%s' '%s'\n",
		typeID, ID, name, title, units, family, context, chartType, priority, updateEvery, options, plugin, module)
	return err
}

//dimension defines a new dimension for the chart.
func (w *apiWriter) dimension(
	ID string,
	name string,
	algorithm dimAlgo,
	multiplier dimDivMul,
	divisor dimDivMul,
	options DimOpts) error {
	_, err := fmt.Fprintf(w, "DIMENSION '%s' '%s' '%s' '%s' '%s' '%s'\n",
		ID, name, algorithm, multiplier, divisor, options)
	return err
}

// begin initialize data collection for a chart.
func (w *apiWriter) begin(typeID string, ID string, msSince int) error {
	var err error
	if msSince > 0 {
		_, err = fmt.Fprintf(w, "BEGIN %s.%s %d\n", typeID, ID, msSince)
	} else {
		_, err = fmt.Fprintf(w, "BEGIN %s.%s\n", typeID, ID)
	}
	return err
}

// dimSet sets the value of a dimension for the initialized chart.
func (w *apiWriter) dimSet(ID string, value int64) error {
	_, err := fmt.Fprintf(w, "SET %s = %d\n", ID, value)
	return err
}

// dimSetEmpty sets the empty value of a dimension for the initialized chart.
func (w *apiWriter) dimSetEmpty(ID string) error {
	_, err := fmt.Fprintf(w, "SET %s = \n", ID)
	return err
}

// varSet sets the value of a variable for the initialized chart.
func (w *apiWriter) varSet(ID string, value int64) error {
	_, err := fmt.Fprintf(w, "VARIABLE CHART %s = %d\n", ID, value)
	return err
}

// end complete data collection for the initialized chart.
func (w *apiWriter) end() error {
	_, err := fmt.Fprintf(w, "END\n\n")
	return err
}

// flush ignore the last collected values.
func (w *apiWriter) flush() error {
	_, err := fmt.Fprintf(w, "FLUSH\n")
	return err
}
