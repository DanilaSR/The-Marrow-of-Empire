// Unit tests for Story SAVE-002: Steam Cloud contract (TR-save-003, ADR-0002
// CR-7). The contract is exercised against the no-op stub and a fake
// implementation — the real Steam Auto-Cloud is an external service with
// bindings the engine does not yet expose, so only the interface contract is
// tested (never the Steamworks API). Test naming:
// Test[Feature]_[Scenario]_[Expected].
package save_test

import (
	"errors"
	"testing"
	"time"

	"marrow-of-empire/save"
)

// AC-3: GIVEN a build without Steam (no `steam` tag; offline/GOG), WHEN Cloud
// operations are attempted, THEN every operation reports the cloud as
// unavailable and never panics — local save/load semantics are unaffected.
func TestNewCloud_WithoutSteam_ReportsCloudUnavailable(t *testing.T) {
	cloud := save.NewCloud()
	if cloud.Available() {
		t.Error("NewCloud().Available() = true, want false without Steam")
	}
	if err := cloud.RegisterFile("save_0.sav", []byte("data")); !errors.Is(err, save.ErrCloudUnavailable) {
		t.Errorf("RegisterFile error = %v, want ErrCloudUnavailable", err)
	}
	info, err := cloud.FileInfo("save_0.sav")
	if !errors.Is(err, save.ErrCloudUnavailable) {
		t.Errorf("FileInfo error = %v, want ErrCloudUnavailable", err)
	}
	if info.Name != "save_0.sav" || info.Exists {
		t.Errorf("FileInfo = %+v, want name echoed with Exists=false", info)
	}
	if data, err := cloud.ReadFile("save_0.sav"); data != nil || !errors.Is(err, save.ErrCloudUnavailable) {
		t.Errorf("ReadFile = %v, %v, want nil, ErrCloudUnavailable", data, err)
	}
	if err := cloud.DeleteFile("save_0.sav"); !errors.Is(err, save.ErrCloudUnavailable) {
		t.Errorf("DeleteFile error = %v, want ErrCloudUnavailable", err)
	}
}

// fakeCloud is a stand-in for the future Steam backend: it implements the
// Cloud contract and returns canned cloud-copy data. The compile-time
// assertion below pins the contract shape so the integration point stays
// implementable.
type fakeCloud struct {
	file save.CloudFile
}

var _ save.Cloud = (*fakeCloud)(nil)

func (f *fakeCloud) Available() bool { return true }

func (f *fakeCloud) RegisterFile(name string, data []byte) error { return nil }

func (f *fakeCloud) FileInfo(name string) (save.CloudFile, error) {
	info := f.file
	info.Name = name
	return info, nil
}

func (f *fakeCloud) ReadFile(name string) ([]byte, error) { return []byte("cloud data"), nil }

func (f *fakeCloud) DeleteFile(name string) error { return nil }

// AC-2: the Cloud contract carries the data the conflict dialog needs — the
// cloud copy's timestamp (compared against the local copy at launch) and
// existence flag. The dialog itself is UI and is not tested here.
func TestCloud_FileInfoCarriesConflictData(t *testing.T) {
	wantTime := time.Date(2026, 8, 9, 12, 30, 0, 0, time.UTC)
	cloud := &fakeCloud{file: save.CloudFile{
		Exists:     true,
		Size:       4096,
		ModifiedAt: wantTime,
	}}

	info, err := cloud.FileInfo("save_2.sav")
	if err != nil {
		t.Fatalf("FileInfo error: %v", err)
	}
	if !info.Exists {
		t.Error("FileInfo().Exists = false, want true (cloud copy present)")
	}
	if info.Size != 4096 {
		t.Errorf("FileInfo().Size = %d, want 4096", info.Size)
	}
	if !info.ModifiedAt.Equal(wantTime) {
		t.Errorf("FileInfo().ModifiedAt = %v, want %v (compared against the local copy)", info.ModifiedAt, wantTime)
	}
}
