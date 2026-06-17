package medical

import "testing"

func TestIsCommercialPopup(t *testing.T) {
	commercialTypes := []PopupType{PopupCommercial, PopupPromotion, PopupUpsell, PopupAd}
	for _, pt := range commercialTypes {
		if !IsCommercialPopup(pt) {
			t.Errorf("IsCommercialPopup(%q) = false, want true", pt)
		}
	}

	nonCommercialTypes := []PopupType{PopupSystem, PopupMedical}
	for _, pt := range nonCommercialTypes {
		if IsCommercialPopup(pt) {
			t.Errorf("IsCommercialPopup(%q) = true, want false", pt)
		}
	}
}

func TestPopupTypeConstants(t *testing.T) {
	types := []PopupType{
		PopupCommercial, PopupPromotion, PopupUpsell, PopupAd,
		PopupSystem, PopupMedical,
	}
	if len(types) != 6 {
		t.Errorf("PopupType count = %d, want 6", len(types))
	}
}

func TestPopupBlockerCreation(t *testing.T) {
	blocker := NewPopupBlocker(nil)
	if blocker == nil {
		t.Fatal("NewPopupBlocker should not return nil")
	}
}
