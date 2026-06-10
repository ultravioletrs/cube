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
}

func TestSourceProviderAliases(t *testing.T) {
	aliases := SourceProviderAliases()
	if aliases[SourceTypeOneDrive] != SourceTypeMicrosoft {
		t.Fatalf("expected onedrive alias to microsoft, got %q", aliases[SourceTypeOneDrive])
	}
	if aliases[SourceTypeSharePoint] != SourceTypeMicrosoft {
		t.Fatalf("expected sharepoint alias to microsoft, got %q", aliases[SourceTypeSharePoint])
	}
}

func TestHumanSourceTypeList(t *testing.T) {
	got := HumanSourceTypeList([]SourceType{SourceTypeGoogleDrive, SourceTypeS3, SourceTypeMicrosoft})
	if got != "google_drive, s3 or microsoft" {
		t.Fatalf("unexpected list format: %q", got)
	}
}
