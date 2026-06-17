package dogwalk

import "testing"

func TestCheckAutoOrderBlocked(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.CheckAutoOrder(false)
	if !dec.Blocked {
		t.Error("auto order without user confirmation should be blocked")
	}
	if dec.Action != RiskActionAutoOrder {
		t.Errorf("Action = %q, want %q", dec.Action, RiskActionAutoOrder)
	}
}

func TestCheckAutoOrderAllowed(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.CheckAutoOrder(true)
	if dec.Blocked {
		t.Error("user-confirmed order should be allowed")
	}
}

func TestCheckMarketingBlocked(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.CheckMarketing("您的宠物可能有健康风险，推荐购买体检套餐", false)
	if !dec.Blocked {
		t.Error("forced marketing should be blocked")
	}
	if dec.Action != RiskActionForceMarket {
		t.Errorf("Action = %q, want %q", dec.Action, RiskActionForceMarket)
	}
}

func TestCheckMarketingAllowed(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.CheckMarketing("今日遛狗服务推荐", true)
	if dec.Blocked {
		t.Error("user-triggered marketing should be allowed")
	}
}

func TestCheckVendorSortBlocked(t *testing.T) {
	gw := NewRiskGateway()

	blockedFields := []string{"paid_score", "internal_rank", "ad_weight"}
	for _, field := range blockedFields {
		dec := gw.CheckVendorSort(field)
		if !dec.Blocked {
			t.Errorf("sort by %q should be blocked", field)
		}
		if dec.Action != RiskActionHiddenWeight {
			t.Errorf("Action = %q, want %q", dec.Action, RiskActionHiddenWeight)
		}
	}
}

func TestCheckVendorSortAllowed(t *testing.T) {
	gw := NewRiskGateway()

	allowedFields := []string{"rating", "distance", "price", "completion", "responseTime"}
	for _, field := range allowedFields {
		dec := gw.CheckVendorSort(field)
		if dec.Blocked {
			t.Errorf("sort by %q should be allowed", field)
		}
	}
}

func TestCheckPaidPlacementBlocked(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.CheckPaidPlacement("vendor_001", true)
	if !dec.Blocked {
		t.Error("paid placement should be blocked")
	}
	if dec.Action != RiskActionPaidTop {
		t.Errorf("Action = %q, want %q", dec.Action, RiskActionPaidTop)
	}
}

func TestCheckPaidPlacementAllowed(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.CheckPaidPlacement("vendor_001", false)
	if dec.Blocked {
		t.Error("non-paid vendor should be allowed")
	}
}

func TestValidateOrderCreationAutoOrder(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.ValidateOrderCreation(true, true, false)
	if !dec.Blocked {
		t.Error("auto order should be blocked even with user confirmation")
	}
}

func TestValidateOrderCreationMarketingDriven(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.ValidateOrderCreation(true, false, true)
	if !dec.Blocked {
		t.Error("marketing-driven order should be blocked")
	}
}

func TestValidateOrderCreationNotConfirmed(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.ValidateOrderCreation(false, false, false)
	if !dec.Blocked {
		t.Error("unconfirmed order should be blocked")
	}
}

func TestValidateOrderCreationValid(t *testing.T) {
	gw := NewRiskGateway()

	dec := gw.ValidateOrderCreation(true, false, false)
	if dec.Blocked {
		t.Error("valid order creation should be allowed")
	}
}

func TestNewRiskGatewayDefaults(t *testing.T) {
	gw := NewRiskGateway()

	if len(gw.blockedActions) != 4 {
		t.Errorf("blockedActions count = %d, want 4", len(gw.blockedActions))
	}
}
