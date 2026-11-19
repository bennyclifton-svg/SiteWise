package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5"
	maildata "sitewise/data/mail"
	"sitewise/internal/mailin"
)

// Record fields worked out when a message is recorded
// (docs/plans/2026-11-17-correspondence-transmittals.md, Task 1): its
// thread, and the firm and package it belongs to. Go only; no Jev.

// exactOnly are free-mail and shared sending domains (data/mail/
// domains.json): a sender there matches a firm only by its exact address,
// never by its domain (review LOW-2).
var exactOnly = func() []string {
	var f struct{ Free, Shared []string }
	if err := json.Unmarshal(maildata.Domains, &f); err != nil {
		panic("data/mail/domains.json: " + err.Error())
	}
	return append(f.Free, f.Shared...)
}()

// FreeMail reports a domain where an address says nothing about a firm:
// free mail, or a shared sending service. Each entry matches its
// subdomains; one ending in a dot matches every domain it starts.
func FreeMail(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return false
	}
	for _, e := range exactOnly {
		if strings.HasSuffix(e, ".") {
			if strings.HasPrefix(domain, e) {
				return true
			}
			continue
		}
		if domain == e || strings.HasSuffix(domain, "."+e) {
			return true
		}
	}
	return false
}

type mailLink struct {
	Thread, Firm, Package, By string
}

// recordLinks threads a message and links its firm and package:
//
//   - Thread: In-Reply-To, then References last to first, against this
//     project's message ids; the first found gives the thread, else the
//     message starts its own. A reply that arrives before its parent starts
//     its own thread and is not re-threaded later.
//   - Firm: an exact address match among the org's firms, else the one firm
//     whose email domain is the address's (never a free-mail domain). For
//     In the sender; for Out the first recipient that matches.
//   - Package: that firm's one package on this project (invited, quoted or
//     awarded); several leave it blank for the PM; no firm takes the
//     thread's package.
func recordLinks(ctx context.Context, tx pgx.Tx, org, project, self, direction, sender, recipients, inReplyTo, references string) (mailLink, error) {
	link := mailLink{Thread: self}
	threadPackage := ""
	ids := strings.Fields(inReplyTo)
	refs := strings.Fields(references)
	for i := len(refs) - 1; i >= 0; i-- {
		ids = append(ids, refs[i])
	}
	if len(ids) > 0 {
		var thread, pkg string
		err := tx.QueryRow(ctx, `SELECT m.thread_id::text, COALESCE(m.package_id::text,'') FROM unnest($3::text[]) WITH ORDINALITY AS w(id,ord)
JOIN mail_messages m ON m.org_id=$1::uuid AND m.project_id=$2::uuid AND m.message_id=w.id
ORDER BY w.ord LIMIT 1`, org, project, ids).Scan(&thread, &pkg)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return link, err
		}
		if err == nil {
			link.Thread, threadPackage = thread, pkg
		}
	}
	addrs := []string{sender}
	if direction == "out" {
		addrs = addressList(recipients)
	}
	firm, err := firmForAddresses(ctx, tx, org, addrs)
	if err != nil {
		return link, err
	}
	link.Firm = firm
	if firm != "" {
		pkgs, err := firmPackages(ctx, tx, org, project, firm)
		if err != nil {
			return link, err
		}
		if len(pkgs) == 1 {
			link.Package, link.By = pkgs[0], "rule"
		}
		return link, nil
	}
	if threadPackage != "" {
		link.Package, link.By = threadPackage, "rule"
	}
	return link, nil
}

// addressList reads a To or Cc header into lower-case addresses.
func addressList(header string) []string {
	var out []string
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if a, err := mail.ParseAddress(part); err == nil {
			out = append(out, strings.ToLower(a.Address))
		} else if strings.Contains(part, "@") && !strings.ContainsAny(part, " <>") {
			out = append(out, strings.ToLower(part))
		}
	}
	return out
}

func domainOf(addr string) string {
	_, d, _ := strings.Cut(strings.ToLower(addr), "@")
	return d
}

// firmForAddresses is the org's firm for the first address that names one.
func firmForAddresses(ctx context.Context, tx pgx.Tx, org string, addrs []string) (string, error) {
	for _, a := range addrs {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		var firm string
		err := tx.QueryRow(ctx, `SELECT id::text FROM firms WHERE org_id=$1::uuid AND email<>'' AND lower(email)=$2 ORDER BY created_at LIMIT 1`, org, a).Scan(&firm)
		if err == nil {
			return firm, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
		d := domainOf(a)
		if d == "" || FreeMail(d) {
			continue
		}
		rows, err := tx.Query(ctx, `SELECT id::text FROM firms WHERE org_id=$1::uuid AND email<>'' AND lower(split_part(email,'@',2))=$2 LIMIT 2`, org, d)
		if err != nil {
			return "", err
		}
		var found []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return "", err
			}
			found = append(found, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return "", err
		}
		if len(found) == 1 {
			return found[0], nil
		}
	}
	return "", nil
}

// firmPackages are the firm's packages on this project: invited, quoted or
// awarded (an award is on a quote).
func firmPackages(ctx context.Context, tx pgx.Tx, org, project, firm string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT package_id::text FROM package_invitations WHERE org_id=$1::uuid AND project_id=$2::uuid AND firm_id=$3::uuid
UNION SELECT package_id::text FROM submissions WHERE org_id=$1::uuid AND project_id=$2::uuid AND firm_id=$3::uuid AND package_id IS NOT NULL
ORDER BY 1 LIMIT 3`, org, project, firm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// MailOutsideAdmitted reports whether mail from someone outside the org is
// filed rather than held (correspondence plan, A1 and owner decision 1): the
// sender's address is exactly a firm's on this project (invited, quoted or
// awarded), or the message replies (In-Reply-To or References) to one
// SiteWise sent for this project AND that send went to this sender: its
// exact address, or failing that its registrable domain when that is not a
// free-mail domain (security review M1). A firm's domain alone never admits.
// The caller has already checked the sender is authenticated.
func (s *Store) MailOutsideAdmitted(ctx context.Context, org, project, sender, inReplyTo, references string) (bool, error) {
	if !validUUID(org) || !validUUID(project) {
		return false, ErrNotFound
	}
	sender = strings.ToLower(strings.TrimSpace(sender))
	var firm bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM firms f WHERE f.org_id=$1::uuid AND f.email<>'' AND lower(f.email)=$3
  AND (EXISTS(SELECT 1 FROM package_invitations i WHERE i.org_id=f.org_id AND i.project_id=$2::uuid AND i.firm_id=f.id)
    OR EXISTS(SELECT 1 FROM submissions u WHERE u.org_id=f.org_id AND u.project_id=$2::uuid AND u.firm_id=f.id)))`,
		org, project, sender).Scan(&firm); err != nil || firm {
		return firm, err
	}
	ids := append(strings.Fields(inReplyTo), strings.Fields(references)...)
	if len(ids) == 0 {
		return false, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT to_addrs || ',' || cc_addrs FROM mail_messages
WHERE org_id=$1::uuid AND project_id=$2::uuid AND direction='out' AND number IS NOT NULL AND message_id = ANY($3::text[])`, org, project, ids)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	domain := domainOf(sender)
	for rows.Next() {
		var to string
		if err := rows.Scan(&to); err != nil {
			return false, err
		}
		for _, a := range addressList(to) {
			if a == sender {
				return true, nil
			}
			if !FreeMail(domain) && mailin.SameRegistrable(domainOf(a), domain) {
				return true, nil
			}
		}
	}
	return false, rows.Err()
}

// MailContact is a firm of this project with an email, as the project's own
// records hold it.
type MailContact struct{ Name, Email string }

// ProjectMailContacts are the firms invited, quoted or awarded on this
// project that have an email (review MEDIUM-4).
func (s *Store) ProjectMailContacts(ctx context.Context, org, project string) ([]MailContact, error) {
	if !validUUID(org) || !validUUID(project) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT f.name, lower(f.email) FROM firms f
WHERE f.org_id=$1::uuid AND f.email<>''
  AND (EXISTS(SELECT 1 FROM package_invitations i WHERE i.org_id=f.org_id AND i.project_id=$2::uuid AND i.firm_id=f.id)
    OR EXISTS(SELECT 1 FROM submissions u WHERE u.org_id=f.org_id AND u.project_id=$2::uuid AND u.firm_id=f.id))`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MailContact
	for rows.Next() {
		var c MailContact
		if err := rows.Scan(&c.Name, &c.Email); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
