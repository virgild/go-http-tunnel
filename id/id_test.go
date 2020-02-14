package id_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/virgild/go-http-tunnel/id"
)

func ExampleID_UnmarshalText() {
	data := []byte("Test data")
	testID := id.New(data)
	fmt.Println(testID)
	// Output: 4J6IEFF-6RN6PLS-PGMPQEC-I7R4WD7-AVCSSI5-YPWGGLZ-74TXT55-I5PZXQL
}

func TestID_Blank(t *testing.T) {
	blankID := id.New(nil)
	require.Equal(t, "4OYMIQU-Y7QOBJ5-GX36TEJ-S35ZEQN-T24QPEM-SNZGTFO-ESWMRW6-CSXBKQM", blankID.String())
}

func TestID_Back(t *testing.T) {
	testID := id.New(nil)
	err := testID.UnmarshalText([]byte("4J6IEFF-6RN6PLS-PGMPQEC-I7R4WD7-AVCSSI5-YPWGGLZ-74TXT55-I5PZXQL"))
	require.NoError(t, err)

	refID := id.New([]byte("Test data"))
	require.True(t, testID.Equals(refID))
}
