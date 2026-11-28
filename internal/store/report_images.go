package store

import (
	"context"
	"encoding/hex"
	"errors"
	"io"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/export"
	"sitewise/internal/files"
	"sitewise/internal/reports"
)

// Images in the project report's cells (project report plan, slice 5;
// Lane C). A PNG or JPEG the PM drops is stored by content hash and listed
// against the report, so only that report serves it. It is not filed as a
// document and Jev never reads it.

// ReportImageMaxBytes caps one image.
const ReportImageMaxBytes = 8 << 20

// ErrReportImage is a file that is not a PNG or JPEG image this report
// may hold.
var ErrReportImage = errors.New("not a report image")

// SaveReportImage lists stored bytes against a project report. The caller
// has decoded them and stored the blob.
func (s *Store) SaveReportImage(ctx context.Context, org, report, actor string, sum []byte, size int64, mediaType string) error {
	if mediaType != "image/png" && mediaType != "image/jpeg" || size < 1 || size > ReportImageMaxBytes {
		return ErrReportImage
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	r, err := readReport(ctx, tx, org, report)
	if err != nil {
		return err
	}
	if r.Kind != reports.KindMonthly {
		return ErrNotFound
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO report_images(org_id,project_id,report_id,sha256,media_type,byte_size,created_by) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7::uuid) ON CONFLICT DO NOTHING`,
		org, r.ProjectID, r.ID, sum, mediaType, size, actor); err != nil {
		return err
	}
	// Backup and restore account for every stored blob through files.
	if _, err := tx.Exec(ctx, `INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6) ON CONFLICT(org_id,project_id,sha256) DO NOTHING`,
		org, newID(), r.ProjectID, sum, size, mediaType); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ReportImage is one image this report holds: its bytes' hash and type.
// Another report's, or another org's, is ErrNotFound.
func (s *Store) ReportImage(ctx context.Context, org, report, hexSum string) ([]byte, string, error) {
	sum, err := hex.DecodeString(hexSum)
	if err != nil || len(sum) != 32 || !validUUID(org) || !validUUID(report) {
		return nil, "", ErrNotFound
	}
	var media string
	err = s.pool.QueryRow(ctx, `SELECT media_type FROM report_images WHERE org_id=$1::uuid AND report_id=$2::uuid AND sha256=$3`, org, report, sum).Scan(&media)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	return sum, media, err
}

// PlaceReportImage lists a filed image document of the report's project
// against the report, and returns its hash in hex. A document that is not a
// PNG or JPEG is ErrReportImage.
func (s *Store) PlaceReportImage(ctx context.Context, org, report, actor, document string) (string, string, error) {
	if !validUUID(document) {
		return "", "", ErrNotFound
	}
	var project string
	if err := s.pool.QueryRow(ctx, `SELECT project_id::text FROM reports WHERE org_id=$1::uuid AND id=$2::uuid AND kind='monthly'`, org, report).Scan(&project); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", err
	}
	var sum []byte
	var size int64
	var media, name string
	err := s.pool.QueryRow(ctx, `SELECT f.sha256, f.byte_size, f.media_type, d.filename FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id
WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND d.id=$3::uuid`, org, project, document).Scan(&sum, &size, &media, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", err
	}
	if err := s.SaveReportImage(ctx, org, report, actor, sum, size, media); err != nil {
		return "", "", err
	}
	return hex.EncodeToString(sum), name, nil
}

// reportImageBytes reads every image a draft's cells name, for the PDF.
// missing is the first cell whose image is not this report's or whose
// bytes are gone: issue is refused, naming it.
func reportImageBytes(ctx context.Context, tx pgx.Tx, org, report string, sections []reports.Section, blobs *files.Store) (map[string][]byte, string, error) {
	out := map[string][]byte{}
	for _, s := range sections {
		for _, b := range s.Blocks {
			if b.Table == nil || b.Table.Meta == nil {
				continue
			}
			for i, row := range b.Table.Rows {
				for c, cell := range row {
					for _, ref := range export.ImageRefs(cell) {
						if _, ok := out[ref[1]]; ok {
							continue
						}
						where := b.Label
						if i < len(b.Table.Rows) && len(row) > 0 && c < len(b.Table.Columns) {
							where += ", " + row[0] + ", " + b.Table.Columns[c]
						}
						sum, _ := hex.DecodeString(ref[1])
						var listed bool
						if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM report_images WHERE org_id=$1::uuid AND report_id=$2::uuid AND sha256=$3)`, org, report, sum).Scan(&listed); err != nil {
							return nil, "", err
						}
						if !listed || blobs == nil {
							return out, where, nil
						}
						f, err := blobs.Open(sum)
						if err != nil {
							return out, where, nil
						}
						data, err := io.ReadAll(io.LimitReader(f, ReportImageMaxBytes+1))
						f.Close()
						if err != nil {
							return out, where, nil
						}
						out[ref[1]] = data
					}
				}
			}
		}
	}
	return out, "", nil
}

// ImageMissing is an issue refused because a cell's image is not there.
type ImageMissing struct{ Cell string }

func (e ImageMissing) Error() string {
	return "the image in " + e.Cell + " is not available; remove it or drop it again before issue"
}

// ReportImages reads the images a report's sections name, for the draft's
// PDF download. A missing one prints its caption.
func (s *Store) ReportImages(ctx context.Context, org, report string, sections []reports.Section, blobs *files.Store) (map[string][]byte, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	images, _, err := reportImageBytes(ctx, tx, org, report, sections, blobs)
	return images, err
}
