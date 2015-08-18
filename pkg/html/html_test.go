package html

import (
	"bytes"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestFormValues(t *testing.T) {
	form := `
<html>
  <body>
    <form id="formy">
      <input type="text" name="text1" value="textvalue1"/>
      <div> 
        <input type="hidden" name="hidden1" value="hiddenvalue1" />
      </div>
      <input type="text" name="repeat1" value="repeatval1"/>
      <input type="text" name="repeat1" value="repeatval2"/>
      <input type="text" name="repeat1" value="repeatval3"/>
    </form>
  </body>
`
	want := map[string][]string{
		"text1": []string{
			"textvalue1",
		},
		"hidden1": []string{
			"hiddenvalue1",
		},
		"repeat1": []string{
			"repeatval1",
			"repeatval2",
			"repeatval3",
		},
	}

	values, err := FormValues("#formy", bytes.NewBufferString(form))
	if err != nil {
		t.Errorf("expected nil err: %q", err)
	}

	if diff := pretty.Compare(want, values); diff != "" {
		t.Errorf("Compare(want, got) = %v", diff)
	}

}
