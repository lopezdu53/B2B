//nolint:testpackage // bulkTrashSuccess is unexported on the admin package
package admin

import "testing"

func TestBulkTrashSuccess(t *testing.T) {
	t.Parallel()

	if got := bulkTrashSuccess(1); got != "1 negocio enviado a la papelera" {
		t.Fatalf("n=1: got %q", got)
	}

	if got := bulkTrashSuccess(50); got != "50 negocios enviados a la papelera" {
		t.Fatalf("n=50: got %q", got)
	}
}
