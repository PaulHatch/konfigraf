// Package memory is a storage backend base on memory
package sqlstore

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/paulhatch/konfigraf/proxy"

	"database/sql"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
)

var ErrRefHasChanged = fmt.Errorf("reference has changed concurrently")

type Storage struct {
	db           *proxy.DB
	repositoryID int
}

// Module returns a Storer representing a submodule, if not exists returns a
// new empty Storer is returned
func (s *Storage) Module(name string) (storage.Storer, error) {
	return nil, fmt.Errorf("sub modules are not supported by this storer")
}

func NewStorage(db *proxy.DB, repository string) (*Storage, error) {

	row, err := db.QueryRow("SELECT id FROM repository WHERE name = $1", []string{"text"}, repository)
	if err != nil {
		return nil, err
	}

	var repositoryID int
	err = row.Scan(&repositoryID)
	if err != nil {
		return nil, err
	}

	return &Storage{db, repositoryID}, nil
}

func (s *Storage) NewEncodedObject() plumbing.EncodedObject {
	return &plumbing.MemoryObject{}
}

func (s *Storage) SetEncodedObject(obj plumbing.EncodedObject) (plumbing.Hash, error) {

	hash := obj.Hash()
	objType := obj.Type()
	r, err := obj.Reader()
	if err != nil {
		return hash, err
	}

	r.Close()

	c, err := ioutil.ReadAll(r)
	if err != nil {
		return hash, err
	}

	err = s.db.Exec(
		"INSERT INTO objects (repo_id, obj_type, hash, blob) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
		[]string{"integer", "integer", "text", "bytea"},
		s.repositoryID,
		int(objType),
		hash.String(),
		c)

	return hash, err
}

// EncodedObject gets an object by hash with the given
// plumbing.ObjectType. Implementors should return
// (nil, plumbing.ErrObjectNotFound) if an object doesn't exist with
// both the given hash and object type.
//
// Valid plumbing.ObjectType values are CommitObject, BlobObject, TagObject,
// TreeObject and AnyObject. If plumbing.AnyObject is given, the object must
// be looked up regardless of its type.
func (s *Storage) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {

	hash := h.String()
	var objType plumbing.ObjectType
	var content []byte
	var err error

	if t == plumbing.AnyObject {
		row, err := s.db.QueryRow(
			"SELECT obj_type, blob FROM objects WHERE repo_id = $1 AND hash = $2",
			[]string{"integer", "char(40)"},
			s.repositoryID,
			hash)

		if err != nil {
			return nil, plumbing.ErrObjectNotFound
			//return nil, err
		}

		err = row.Scan(&objType, &content)
	} else {
		row, err := s.db.QueryRow(
			"SELECT blob FROM objects WHERE repo_id = $1 AND obj_type = $2 AND hash = $3",
			[]string{"integer", "integer", "char(40)"},
			s.repositoryID,
			int(t),
			hash)

		if err != nil {
			return nil, plumbing.ErrObjectNotFound
			//return nil, err
		}

		objType = t
		err = row.Scan(&content)
	}

	if err != nil {
		//if err == sql.ErrNoRows {
		return nil, plumbing.ErrObjectNotFound
		//}
		//return nil, err
	}

	return newObject(objType, []byte(content))
}

func newObject(t plumbing.ObjectType, content []byte) (plumbing.EncodedObject, error) {
	o := &plumbing.MemoryObject{}
	o.SetType(t)
	o.SetSize(int64(len(content)))

	_, err := o.Write(content)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (s *Storage) IterEncodedObjects(t plumbing.ObjectType) (storer.EncodedObjectIter, error) {

	rows, err := s.db.Query(
		"SELECT blob FROM objects WHERE repo_id = $1 AND obj_type = $2",
		[]string{"integer", "integer"},
		s.repositoryID,
		int(t))

	if err != nil {
		return nil, err
	}

	return &EncodedObjectIter{t, rows}, nil
}

// HasEncodedObject returns ErrObjNotFound if the object doesn't
// exist.  If the object does exist, it returns nil.
func (s *Storage) HasEncodedObject(h plumbing.Hash) error {

	row, err := s.db.QueryRow(
		"SELECT COUNT(*) FROM objects WHERE repo_id = $1 AND hash = $2",
		[]string{"integer", "char(40)"},
		s.repositoryID,
		h.String())

	if err != nil {
		return err
	}

	var r int
	err = row.Scan(&r)

	if err != nil {
		return err
	}

	if r > 0 {
		return nil
	}

	return plumbing.ErrObjectNotFound

}

// EncodedObjectSize returns the plaintext size of the encoded object.
func (s *Storage) EncodedObjectSize(plumbing.Hash) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

type EncodedObjectIter struct {
	t    plumbing.ObjectType
	rows *proxy.Rows
}

func (i *EncodedObjectIter) Close() {
}

func (i *EncodedObjectIter) Next() (plumbing.EncodedObject, error) {
	r := i.rows.Next()
	if !r {
		return nil, io.EOF
	}

	var content []byte
	i.rows.Scan(&content)

	return newObject(i.t, []byte(content))
}

func (i *EncodedObjectIter) ForEach(cb func(obj plumbing.EncodedObject) error) error {
	for {
		obj, err := i.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		if err := cb(obj); err != nil {
			if err == storer.ErrStop {
				return nil
			}

			return err
		}
	}
}

func (s *Storage) SetReference(ref *plumbing.Reference) error {

	raw := ref.Strings()
	err := s.db.Exec(
		"INSERT INTO refs (repo_id, name, target) VALUES ($1,$2,$3) ON CONFLICT ON CONSTRAINT refs_pk DO UPDATE SET target = $3",
		[]string{"integer", "text", "text"},
		s.repositoryID,
		raw[0],
		raw[1])

	return err
}

// CheckAndSetReference sets the reference `new`, but if `old` is
// not `nil`, it first checks that the current stored value for
// `old.Name()` matches the given reference value in `old`.  If
// not, it returns an error and doesn't update `new`.
func (s *Storage) CheckAndSetReference(new, old *plumbing.Reference) error {
	if new == nil {
		return nil
	}

	if old != nil {
		tmp, err := s.Reference(old.Name())
		if err == nil || tmp.Hash() != old.Hash() {
			return ErrRefHasChanged
		}
	}
	return s.SetReference(new)
}

func (s *Storage) Reference(n plumbing.ReferenceName) (*plumbing.Reference, error) {

	row, err := s.db.QueryRow(
		"SELECT name, target FROM refs WHERE repo_id = $1 AND name = $2",
		[]string{"integer", "text"},
		s.repositoryID,
		n.String())

	if err != nil {
		return nil, plumbing.ErrReferenceNotFound
		//return nil, err
	}

	var name, target string
	err = row.Scan(&name, &target)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, plumbing.ErrReferenceNotFound
		}
		return nil, err
	}

	return plumbing.NewReferenceFromStrings(name, target), nil
}

func (s *Storage) IterReferences() (storer.ReferenceIter, error) {

	rows, err := s.db.Query(
		"SELECT name, target FROM refs WHERE repo_id = $1",
		[]string{"integer"},
		s.repositoryID)

	if err != nil {
		return nil, err
	}

	var refs []*plumbing.Reference
	for rows.Next() {
		var name, target string
		err = rows.Scan(&name, &target)
		if err != nil {
			return nil, err
		}

		refs = append(refs, plumbing.NewReferenceFromStrings(
			name,
			target,
		))
	}

	return storer.NewReferenceSliceIter(refs), nil
}

func (s *Storage) RemoveReference(n plumbing.ReferenceName) error {
	// TODO: Change after exec issue is solved
	_, err := s.db.QueryRow(
		"DELETE FROM refs WHERE repo_id = $1 and name = $2 RETURNING 1",
		[]string{"integer", "text"},
		s.repositoryID,
		n.String())

	return err
}

func (s *Storage) CountLooseRefs() (int, error) {
	row, err := s.db.QueryRow(
		"SELECT COUNT(*) FROM refs WHERE repo_id = $1",
		[]string{"integer"},
		s.repositoryID)

	if err != nil {
		return 0, err
	}

	var r int
	return r, row.Scan(&r)
}

func (s *Storage) PackRefs() error {
	return nil
}

// Config gets the config
func (s *Storage) Config() (*config.Config, error) {

	row, err := s.db.QueryRow("SELECT data FROM config WHERE repo_id = $1", []string{"integer"}, s.repositoryID)
	if err != nil {
		return config.NewConfig(), nil
		//return nil, err
	}

	var data []byte
	err = row.Scan(&data)

	if err != nil {
		if err == sql.ErrNoRows {
			return config.NewConfig(), nil
		}
		return nil, err
	}

	c := &config.Config{
		Remotes:  make(map[string]*config.RemoteConfig),
		Branches: make(map[string]*config.Branch),
	}

	err = c.Unmarshal([]byte(data))

	return c, err
}

func (s *Storage) SetConfig(c *config.Config) error {

	data, err := c.Marshal()
	if err != nil {
		return err
	}

	err = s.db.Exec(
		"INSERT INTO config (repo_id, data) VALUES ($1, $2) ON CONFLICT ON CONSTRAINT config_pk DO UPDATE SET data = $2",
		[]string{"integer", "bytea"},
		s.repositoryID,
		data)

	return err
}

func (s *Storage) Index() (*index.Index, error) {
	row, err := s.db.QueryRow("SELECT data::text FROM index WHERE repo_id = $1", []string{"integer"}, s.repositoryID)
	if err != nil {
		return &index.Index{Version: 2}, nil
		//return nil, err
	}

	var data string
	err = row.Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return &index.Index{Version: 2}, nil
		}
		return nil, err
	}

	idx := &index.Index{}

	return idx, json.Unmarshal([]byte(data), idx)
}

func (s *Storage) SetIndex(idx *index.Index) error {
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}

	err = s.db.Exec(
		"INSERT INTO index (repo_id, data) VALUES ($1, $2::json) ON CONFLICT ON CONSTRAINT index_pk DO UPDATE SET data = $2::json",
		[]string{"integer", "text"},
		s.repositoryID,
		string(data))

	return err
}

func (s *Storage) Shallow() ([]plumbing.Hash, error) {
	row, err := s.db.QueryRow("SELECT data FROM shallow WHERE repo_id = $1", []string{"integer"}, s.repositoryID)
	var h []plumbing.Hash
	if err != nil {
		return h, nil
		//return nil, err
	}

	var data string
	err = row.Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return h, nil
		}
		return nil, err
	}

	return h, json.Unmarshal([]byte(data), &h)
}

func (s *Storage) SetShallow(hash []plumbing.Hash) error {
	data, err := json.Marshal(hash)
	if err != nil {
		return err
	}

	err = s.db.Exec(
		"INSERT INTO shallow (repo_id, data) VALUES ($1, $2::json) ON CONFLICT ON CONSTRAINT shallow_pk DO UPDATE SET data = $2::json",
		[]string{"integer", "text"},
		s.repositoryID,
		string(data))

	return err
}
