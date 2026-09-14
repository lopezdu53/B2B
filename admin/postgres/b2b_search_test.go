package postgres

import "testing"

func TestRestoreTrashOnIngest(t *testing.T) {
	t.Parallel()

	if !restoreTrashOnIngest(false) {
		t.Fatal("first ingest of a search may restore matching leads from the trash")
	}

	if restoreTrashOnIngest(true) {
		t.Fatal("re-ingesting on a Negocios reload must not pull leads back from the trash")
	}
}
