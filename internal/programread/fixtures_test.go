package programread

// sampleHarvest is a small builder's program; sampleTargets are a program's
// rows, one settled by rule ("Pour slab"), the rest open.
func sampleHarvest() Harvest {
	rows := []Row{
		{ID: "r1", Label: "Site establishment", StartRaw: "1/03/2027", FinishRaw: "5/03/2027", DurationRaw: "5 days", Start: "2027-03-01", Finish: "2027-03-05"},
		{ID: "r2", Label: "Excavate footings", StartRaw: "8/03/2027", FinishRaw: "12/03/2027", DurationRaw: "5 days", Start: "2027-03-08", Finish: "2027-03-12"},
		{ID: "r3", Label: "Pour slab", StartRaw: "15/03/2027", FinishRaw: "19/03/2027", DurationRaw: "5 days", Start: "2027-03-15", Finish: "2027-03-19"},
		{ID: "r4", Label: "Frame walls and roof", StartRaw: "22/03/2027", FinishRaw: "16/04/2027", DurationRaw: "4 wks", Start: "2027-03-22", Finish: "2027-04-16"},
		{ID: "r5", Label: "Roof cladding", StartRaw: "19/04/2027", FinishRaw: "23/04/2027", DurationRaw: "5d", Start: "2027-04-19", Finish: "2027-04-23"},
	}
	return Harvest{Filename: "Smith program.xlsx", Rows: rows, Checks: []string{}}
}

func sampleTargets() []Target {
	return []Target{
		{ID: "t-cons", Label: "Construction", Kind: "stage", CatalogueStageID: "construction", Tier: TierStage},
		{ID: "t-slab", Label: "Slab", Kind: "activity", StageLabel: "Construction", Tier: TierFragment},
		{ID: "t-pour", Label: "Pour Slab", Kind: "activity", StageLabel: "Construction", Tier: TierPackage},
		{ID: "t-frame", Label: "Wall framing", Kind: "activity", StageLabel: "Construction", Tier: TierPackage},
		{ID: "t-zz", Label: "Qq", Kind: "hold_point", StageLabel: "Construction", Tier: TierPackage},
	}
}
