package domain

// Ref is the binding's Weapon reference as an operator spells it in active.yaml.
// A Custom binding's installed instance id is already the versioned "id@version"
// identity, so appending WeaponVersion again would corrupt it; a Ranked
// binding's instance id is the bare Weapon id and takes its version.
func (b SlotBinding) Ref() string {
	if b.EffectiveMode() == SlotBindingModeCustom {
		return b.InstalledInstanceID
	}
	return WeaponRef(b.InstalledInstanceID, b.WeaponVersion)
}
