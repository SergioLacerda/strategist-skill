package application

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCompletionObjectAcceptsUnknownFields(t *testing.T) {
	require.NoError(t, ValidateCompletionObject([]byte(`{"request_id":"req","result":"ok","unknown":true}`)))
}

func TestValidateCompletionObjectRejectsMalformedObjects(t *testing.T) {
	for _, raw := range [][]byte{
		{}, []byte("[1]"), []byte(`{"request_id":"a","request_id":"b"}`),
		[]byte(`{"request_id":}`), []byte(`{"request_id":"a"`), []byte(`{"request_id":"a"]`),
	} {
		require.Error(t, ValidateCompletionObject(raw))
	}
}
