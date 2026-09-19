package conformance

import "fmt"

// Validate checks inventory, row identity, and evidence-tier invariants.
func (m Matrix) Validate() error {
	if m.SchemaVersion == "" {
		return fmt.Errorf("conformance matrix: schema_version is required")
	}
	clients, err := validateClients(m.Clients)
	if err != nil {
		return err
	}
	return validateRows(m.Rows, clients)
}

func validateClients(values []Client) (map[string]Client, error) {
	clients := make(map[string]Client, len(values))
	for _, client := range values {
		if err := validateClient(client, clients); err != nil {
			return nil, err
		}
		clients[client.ID] = client
	}
	return clients, nil
}

func validateClient(client Client, clients map[string]Client) error {
	if client.ID == "" || client.Owner == "" || client.Surface == "" {
		return fmt.Errorf("conformance matrix: client id, owner, and surface are required")
	}
	if _, exists := clients[client.ID]; exists {
		return fmt.Errorf("conformance matrix: duplicate client %q", client.ID)
	}
	if len(client.Roles) == 0 || len(client.ProviderModes) == 0 || client.UnsupportedPolicy == "" {
		return fmt.Errorf("conformance matrix: client %q has incomplete support policy", client.ID)
	}
	return nil
}

func validateRows(rows []Row, clients map[string]Client) error {
	if len(rows) == 0 {
		return fmt.Errorf("conformance matrix: at least one row is required")
	}
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if err := validateRow(row, clients, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateRow(row Row, clients map[string]Client, seen map[string]struct{}) error {
	if err := validateRowFields(row); err != nil {
		return err
	}
	if _, exists := seen[row.ID]; exists {
		return fmt.Errorf("conformance matrix: duplicate row %q", row.ID)
	}
	seen[row.ID] = struct{}{}
	client, exists := clients[row.Client]
	if !exists {
		return fmt.Errorf("conformance matrix: row %q references unclassified client %q", row.ID, row.Client)
	}
	if err := validateRowInventory(row, client); err != nil {
		return err
	}
	return validateRowEvidence(row)
}

func validateRowFields(row Row) error {
	if row.ID == "" || row.Client == "" || row.Role == "" || row.Slot == "" || row.ProviderMode == "" || row.EnvelopeVersion == "" || row.ReasonCode == "" || row.AuthorityOwner == "" {
		return fmt.Errorf("conformance matrix: row %q is incomplete", row.ID)
	}
	return nil
}

func validateRowInventory(row Row, client Client) error {
	if !contains(client.Roles, row.Role) || !contains(client.ProviderModes, row.ProviderMode) {
		return fmt.Errorf("conformance matrix: row %q is outside client %q inventory", row.ID, row.Client)
	}
	return nil
}

func validateRowEvidence(row Row) error {
	if row.EvidenceTier != EvidenceStructural && row.EvidenceTier != EvidenceLive {
		return fmt.Errorf("conformance matrix: row %q has unknown evidence tier %q", row.ID, row.EvidenceTier)
	}
	if !validState(row.ExpectedState) {
		return fmt.Errorf("conformance matrix: row %q has unknown expected state %q", row.ID, row.ExpectedState)
	}
	return nil
}

func validState(state EvidenceState) bool {
	switch state {
	case StateCertified, StateStale, StateFailed, StateUnknown, StateUnsupported, StateBlocked,
		StateUnavailable, StateUnauthorized, StateTimeout, StateMalformed, StateTeardownFailed:
		return true
	default:
		return false
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
