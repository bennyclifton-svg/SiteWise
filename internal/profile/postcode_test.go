package profile

import "testing"

func TestStateOfAddress(t *testing.T) {
	for _, tc := range []struct{ address, state, postcode string }{
		{"8 Paperbark Lane, Northbridge NSW 2063", "NSW", "2063"},
		{"1 Main St, Fyshwick 2609", "ACT", "2609"},
		{"12 Smith Street, Fitzroy 3065", "VIC", "3065"},
		{"Lot 4 Stuart Highway, Darwin 0800", "NT", "0800"},
		{"Somewhere without a postcode", "", ""},
	} {
		if s, p := stateOfAddress(tc.address); s != tc.state || p != tc.postcode {
			t.Errorf("%q: %s %s, want %s %s", tc.address, s, p, tc.state, tc.postcode)
		}
	}
}

// Kavanagh: the state was read from consultants' office addresses. The site
// address now suggests it, and contradicts a stated state that differs.
func TestSiteAddressSuggestsAndChecksTheState(t *testing.T) {
	address := Row{PartID: "w", Key: "site.address", Value: "8 Paperbark Lane, Northbridge NSW 2063", Band: "amber"}
	rows := withAddressState([]Row{address}, "w")
	if len(rows) != 2 || rows[1].Key != "det.state" || rows[1].Value != "NSW" || rows[1].Band != bandSuggest {
		t.Fatalf("suggested state %+v", rows)
	}
	rows = withAddressState([]Row{address, {PartID: "w", Key: "det.state", Value: "VIC", Band: "amber"}}, "w")
	if rows[1].Value != "VIC" || rows[1].Note == "" {
		t.Fatalf("contradiction not flagged: %+v", rows[1])
	}
	rows = withAddressState([]Row{address, {PartID: "w", Key: "det.state", Value: "NSW", Band: "amber"}}, "w")
	if rows[1].Note != "" || rows[1].Band != "amber" {
		t.Fatalf("agreeing state changed: %+v", rows[1])
	}
	if rows := withAddressState([]Row{{PartID: "w", Key: "site.address", Band: "blank"}}, "w"); len(rows) != 1 {
		t.Fatalf("state invented without an address: %+v", rows)
	}
}
