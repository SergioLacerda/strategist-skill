package treasure

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// promotionPacketSchemaVersion is the only schema_version
// LoadPromotionPackets accepts today, mirroring jewels.yaml/potions.yaml's
// own schema_version gate (readJewelManifest).
const promotionPacketSchemaVersion = "1"

// promotionPacketManifest is the on-disk shape of
// .strategist/promotion-packets.yaml — the monolithic-file convention
// .analysis/refined/20260830-skill-gaps-triage/promotion-packet-design.md
// recommends, mirroring jewels.yaml/potions.yaml (no partition-file support
// yet: the design doc treats that as a future split, not a day-one
// requirement).
type promotionPacketManifest struct {
	SchemaVersion string                   `yaml:"schema_version"`
	Packets       []domain.PromotionPacket `yaml:"packets"`
}

// LoadPromotionPackets reads root/promotion-packets.yaml (root is the
// .strategist directory). Returns (nil, nil) when the file does not exist —
// mirroring LoadJewels' treatment of an absent manifest as "no packets yet
// raised," not an error.
func LoadPromotionPackets(root string) ([]domain.PromotionPacket, error) {
	path := filepath.Join(root, "promotion-packets.yaml")
	raw, err := os.ReadFile(path) //nolint:gosec // G304: promotion-packets path is derived from the selected runtime root
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read promotion-packets.yaml: %w", err)
	}
	var m promotionPacketManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse promotion-packets.yaml: %w", err)
	}
	if m.SchemaVersion != "" && m.SchemaVersion != promotionPacketSchemaVersion {
		return nil, fmt.Errorf("promotion-packets.yaml: unsupported schema_version %q (expected %q)", m.SchemaVersion, promotionPacketSchemaVersion)
	}
	return m.Packets, nil
}

// SavePromotionPackets atomically writes packets to
// root/promotion-packets.yaml (root is the .strategist directory),
// replacing any existing content. Uses the same temp-sibling-then-rename
// atomicity writeFileAtomic already gives jewels.yaml/potions.yaml writes,
// per the design doc's recommendation to reuse rather than reinvent it.
func SavePromotionPackets(root string, packets []domain.PromotionPacket) error {
	m := promotionPacketManifest{SchemaVersion: promotionPacketSchemaVersion, Packets: packets}
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("encode promotion-packets.yaml: %w", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", root, err)
	}
	path := filepath.Join(root, "promotion-packets.yaml")
	if err := writeFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("write promotion-packets.yaml: %w", err)
	}
	return nil
}

// AddPromotionPacket appends packet to root/promotion-packets.yaml, loading
// the existing manifest first. Rejects a duplicate PacketID rather than
// silently overwriting it — a promotion packet is addressed by PacketID, so
// an overwrite would silently discard the previous candidate's content.
func AddPromotionPacket(root string, packet domain.PromotionPacket) error {
	existing, err := LoadPromotionPackets(root)
	if err != nil {
		return err
	}
	for _, p := range existing {
		if p.PacketID == packet.PacketID {
			return fmt.Errorf("promotion packet %q already exists", packet.PacketID)
		}
	}
	return SavePromotionPackets(root, append(existing, packet))
}
