package specifications

// Definitions are the versioned value contract, not product recommendations or
// compliance rules. Each value remains a project assertion with its own source.
type Definition struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Group   string   `json:"group"`
	Type    string   `json:"type"`
	Units   []string `json:"units,omitempty"`
	Version int      `json:"version"`
}

func Definitions() []Definition {
	text := func(key, label, group string) Definition {
		return Definition{Key: key, Label: label, Group: group, Type: "text", Version: 1}
	}
	number := func(key, label, group string, units ...string) Definition {
		return Definition{Key: key, Label: label, Group: group, Type: "number", Units: units, Version: 1}
	}
	return []Definition{
		text("manufacturer", "Manufacturer", "Identity"), text("model", "Model / product code", "Identity"), text("material", "Material", "Identity"), text("grade", "Grade / species", "Identity"), text("classification", "Classification reference", "Identity"),
		text("colour", "Colour", "Finish"), text("finish", "Finish", "Finish"), text("sheen", "Sheen", "Finish"), text("grain", "Grain / cut / direction", "Finish"), text("edge", "Edge / profile", "Finish"), text("pattern", "Laying pattern", "Finish"), text("grout", "Grout", "Finish"),
		number("width", "Product width", "Dimensions", "mm", "m"), number("height", "Product height", "Dimensions", "mm", "m"), number("depth", "Product depth", "Dimensions", "mm", "m"), number("thickness", "Thickness", "Dimensions", "mm", "micron"), number("length", "Length", "Dimensions", "mm", "m"), number("opening_width", "Opening width", "Dimensions", "mm", "m"), number("opening_height", "Opening height", "Dimensions", "mm", "m"), number("clearance", "Clearance", "Dimensions", "mm", "m"), number("mounting_height", "Mounting height", "Dimensions", "mm", "m"), text("stock_size", "Stock size", "Dimensions"), text("cut_size", "Cut size", "Dimensions"), text("handing", "Handing", "Dimensions"),
		text("performance", "Performance requirement / claim", "Performance"), text("slip_rating", "Slip classification", "Performance"), text("fire_rating", "Fire performance", "Performance"), text("acoustic_rating", "Acoustic performance", "Performance"), text("ip_rating", "IP rating", "Performance"), text("wels_rating", "WELS rating / registration", "Performance"), number("flow", "Flow rate", "Performance", "L/min"), number("power", "Power", "Performance", "W", "kW", "W/m"), number("colour_temperature", "Colour temperature", "Performance", "K"), number("capacity", "Capacity", "Performance", "L", "kg", "kW"),
		text("substrate", "Substrate", "Installation"), text("preparation", "Preparation", "Installation"), text("installation", "Installation method", "Installation"), text("fixing", "Fixing", "Installation"), text("tolerance", "Tolerance and measurement span", "Installation"), text("controls", "Controls / driver", "Installation"), text("service_interface", "Service interface", "Installation"), text("protection", "Protection / make-good", "Installation"), text("existing_condition", "Existing condition", "Installation"),
		text("sample", "Sample / prototype", "Quality"), text("approval", "Approval / hold point", "Quality"), text("installer", "Installer / credential", "Quality"), text("test", "Test / commissioning", "Quality"), text("warranty", "Warranty requirement / offer", "Quality"), text("maintenance", "Maintenance / care", "Quality"), text("dependency", "Drawing / consultant dependency", "Quality"),
		text("supplier", "Supplier", "Commercial"), text("quote", "Quote and commercial basis", "Commercial"), text("inclusions", "Inclusions / exclusions", "Commercial"), text("notes", "Other source information", "Notes"),
	}
}

func DefinitionFor(key string) (Definition, bool) {
	for _, d := range Definitions() {
		if d.Key == key {
			return d, true
		}
	}
	return Definition{}, false
}
