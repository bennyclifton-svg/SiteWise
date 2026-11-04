package store

import "context"

// AppointedFirm is a package awarded to a firm, with the firm's contact as
// the directory holds it (Walsh review, 2026-10-27: the architect's name and
// contact were read on award but shown nowhere on the Profile).
type AppointedFirm struct {
	PackageID string `json:"package_id"`
	Package   string `json:"package"`
	Firm      string `json:"firm"`
	Contact   string `json:"contact,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	// PIExpiresOn is the professional indemnity period end the awarded
	// letter states, or "".
	PIExpiresOn string `json:"pi_expires_on,omitempty"`
}

// AppointedFirms lists the project's live packages with an awarded firm, in
// title order. One indexed read; nothing is asked of Jev.
func (s *Store) AppointedFirms(ctx context.Context, org, project string) ([]AppointedFirm, error) {
	rows, err := s.pool.Query(ctx, `SELECT p.id::text, p.title, f.name, f.contact, f.email, f.phone, COALESCE(s.pi_expires_on::text,'')
FROM packages p
JOIN submissions s ON s.org_id=p.org_id AND s.project_id=p.project_id AND s.package_id=p.id AND s.status='awarded'
JOIN firms f ON f.org_id=s.org_id AND f.id=s.firm_id
WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND p.retired_at IS NULL
ORDER BY p.title, f.name`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AppointedFirm{}
	for rows.Next() {
		var a AppointedFirm
		if err := rows.Scan(&a.PackageID, &a.Package, &a.Firm, &a.Contact, &a.Email, &a.Phone, &a.PIExpiresOn); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
