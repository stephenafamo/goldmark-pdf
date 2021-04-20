package pdf

var FontHelvetica = Font{
	CanUseForText: true,
	CanUseForCode: false,
	Category:      "sans-serif",
	Family:        "Helvetica",
	Type:          FontTypeInbuilt,
}

var FontTimes = Font{
	CanUseForText: true,
	CanUseForCode: false,
	Category:      "serif",
	Family:        "Times",
	Type:          FontTypeInbuilt,
}

var FontCourier = Font{
	CanUseForText: true,
	CanUseForCode: true,
	Category:      "monospace",
	Family:        "Courier",
	Type:          FontTypeInbuilt,
}
