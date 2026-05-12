// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package domain

import "testing"

func TestSourceTypeHelpers(t *testing.T) {
	if !IsSupportedSourceType(SourceTypeLocalFS) {
		t.Fatal("expected local_fs to be supported for service layer")
	}
	if !IsUserCreatableSourceType(SourceTypeS3) {
		t.Fatal("expected s3 to be user-creatable")
	}
	if IsUserCreatableSourceType(SourceTypeLocalFS) {
		t.Fatal("expected local_fs to be non-user-creatable")
	}
	if !IsRcloneBackedSourceType(SourceTypeRclone) || !IsRcloneBackedSourceType(SourceTypeDropbox) {
		t.Fatal("expected rclone and dropbox to be rclone-backed")
	}
	if IsRcloneBackedSourceType(SourceTypeMicrosoft) {
		t.Fatal("did not expect microsoft to be rclone-backed")
	}
}

func TestSourceProviderAliases(t *testing.T) {
	aliases := SourceProviderAliases()
	if aliases[SourceTypeOneDrive] != SourceTypeMicrosoft {
		t.Fatalf("expected onedrive alias to microsoft, got %q", aliases[SourceTypeOneDrive])
	}
	if aliases[SourceTypeSharePoint] != SourceTypeMicrosoft {
		t.Fatalf("expected sharepoint alias to microsoft, got %q", aliases[SourceTypeSharePoint])
	}
	if aliases[SourceTypeDropbox] != SourceTypeRclone {
		t.Fatalf("expected dropbox alias to rclone, got %q", aliases[SourceTypeDropbox])
	}
}

func TestHumanSourceTypeList(t *testing.T) {
	got := HumanSourceTypeList([]SourceType{SourceTypeGoogleDrive, SourceTypeS3, SourceTypeRclone})
	if got != "google_drive, s3 or rclone" {
		t.Fatalf("unexpected list format: %q", got)
	}
}
