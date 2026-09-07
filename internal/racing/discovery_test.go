package racing

import "testing"

func TestBuildDiscoveryConfigs(t *testing.T) {
	configs := BuildDiscoveryConfigs("home/racing", []string{
		"wec", "imsa-sportscar-championship", "wrc", "supercars-championship",
	})
	if len(configs) != 5 {
		t.Fatalf("got %d configs, want 5 (live + 4 series)", len(configs))
	}
	supercars := configs[4]
	if supercars.ObjectID != deviceUID+"_next_supercars_championship" {
		t.Fatalf("object id %q", supercars.ObjectID)
	}
	tmpl, _ := supercars.Payload["value_template"].(string)
	want := "{{ value_json.next_by_series['Supercars Championship'].start }}"
	if tmpl != want {
		t.Fatalf("template %q want %q", tmpl, want)
	}
}
