package service

import (
	"archive/tar"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/filemode"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/paulhatch/konfigraf/proxy"
	"github.com/paulhatch/konfigraf/sqlstore"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

// ErrorType for service errors
type ErrorType int

const (
	// Invalid error status
	Invalid ErrorType = 0
	// NotFound error status
	NotFound ErrorType = 1
	// Conflict error status
	Conflict ErrorType = 2
	// RemoteCredentials error status
	RemoteCredentials ErrorType = 3
	// InvalidReference error status
	InvalidReference ErrorType = 4
	// Remote error status
	Remote ErrorType = 5
	// InvalidArgument error status
	InvalidArgument = 6
)

// Error for service errors
type Error struct {
	err  string
	Code ErrorType
}

func (e Error) Error() string {
	return e.err
}

var errInvalid = &Error{"invalid configuration", Invalid}
var errDoesNotExist = &Error{"configuration doesn't exist", NotFound}
var errFileDoesNotExist = &Error{"file doesn't exist", NotFound}
var errHashConflict = &Error{"hash conflict", Conflict}
var errRemoteCredentials = &Error{"could not access remote credentials", RemoteCredentials}
var errInvalidReference = &Error{"could not resolve reference", InvalidReference}

//var errRemote = &Error{"could push to remote", Remote}
var errBadDB = &Error{"invalid database argument", InvalidArgument}

// CreateRepository registers a new configuration repository
func CreateRepository(db *proxy.DB, name string) (int, error) {

	if len(name) == 0 || len(name) > 50 {
		return 0, errInvalid
	}

	stmt, err := db.QueryRow(`INSERT INTO repository (name) VALUES ($1) RETURNING ID`, []string{"text"}, name)
	if err != nil {
		return 0, err
	}

	var id int
	err = stmt.Scan(&id)
	if err != nil {
		return 0, err
	}

	// if len(remoteURL) > 0 {
	// 	secrets := map[string]interface{}{"data": map[string]interface{}{"user": user, "token": token}}
	// 	_, err = v.Logical().Write(fmt.Sprintf("/secret/data/%s/credentials", name), secrets)

	// 	if err != nil {
	// 		tx.Rollback()
	// 		return err
	// 	}
	// }

	_, err = createRepo(db, name)
	if err != nil {
		return 0, err
	}

	//logrus.WithFields(logrus.Fields{
	//	"repository": name,
	//}).Info("Created repository")

	return id, nil
}

// UpdateRepository sets a repository's remote settings
func UpdateRepository(db *proxy.DB, name string, user string, token string, remoteURL string) error {

	re := regexp.MustCompile("^[a-z0-9_-]+$")

	if len(name) == 0 || len(name) > 50 || !re.Match([]byte(name)) || len(remoteURL) == 0 {
		return errInvalid
	}

	repo, err := openRepo(db, false, name)
	if err != nil {
		return err
	}

	// First remove the remote if it already exists
	_, err = repo.Remote("origin")
	switch err {
	case git.ErrRemoteNotFound:
		break
	case nil:
		err = repo.DeleteRemote("origin")
		if err != nil {
			return err
		}
		break
	default:
		return err
	}

	//_, err = repo.CreateRemote(&config.RemoteConfig{
	//	Name:  "origin",
	//	URLs:  []string{remoteURL},
	//	Fetch: []config.RefSpec{"refs/heads/master:refs/remotes/origin/master"},
	//})
	//if err != nil {
	//	return err
	//}

	// update secrets in vault
	//secrets := map[string]interface{}{"data": map[string]interface{}{"user": user, "token": token}}
	//_, err = v.Logical().Write(fmt.Sprintf("/secret/data/%s/credentials", name), secrets)
	//if err != nil {
	//	tx.Rollback()
	//	return err
	//}

	//logrus.WithFields(logrus.Fields{
	//	"repository": name,
	//}).Info("Updated repository")

	return nil
}

// RepositoryInfo result
type RepositoryInfo struct {
	Remotes map[string][]string
}

// GetRepository returns information about the requested repo
func GetRepository(db *proxy.DB, name string) (*RepositoryInfo, error) {
	if len(name) == 0 || len(name) > 50 {
		return nil, errInvalid
	}

	repo, err := openRepo(db, false, name)
	if err != nil {
		return nil, err
	}

	result := &RepositoryInfo{
		Remotes: make(map[string][]string),
	}

	remotes, err := repo.Remotes()

	if err != nil {
		return nil, err
	}

	for index := 0; index < len(remotes); index++ {
		r := remotes[index].Config()
		result.Remotes[r.Name] = r.URLs
	}

	return result, nil

}

// DeleteRepository removes a configuration repository
func DeleteRepository(db *proxy.DB, name string) error {

	if len(name) == 0 || len(name) > 50 {
		return errInvalid
	}

	return db.Exec(`DELETE FROM repository WHERE name = $1`, []string{"text"}, name)
}

// Repository contains information about a repository
type Repository struct {
	Name string `json:"name"`
}

// GetRepositories lists all repositories
func GetRepositories(db *proxy.DB) ([]*Repository, error) {

	rows, err := db.Query(`SELECT name FROM repository ORDER BY name`, nil)
	if err != nil {
		return nil, err
	}

	repos := []*Repository{}
	for rows.Next() {
		repo := &Repository{}
		err := rows.Scan([]interface{}{&repo.Name})
		if err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}

	return repos, nil
}

// FileInfo represents information about a file within a commit
type FileInfo struct {
	Contents string
	RepoHash string
	ItemHash string
}

// GetFile retrieves a specific file from a repository
func GetFile(
	db *proxy.DB,
	name string,
	branch string,
	path string) (*FileInfo, error) {

	path = strings.TrimPrefix(path, "/")

	repo, err := openRepo(db, false, name)
	if err != nil {
		return nil, err
	}

	hash, err := resolveHashFromName(repo, branch)
	if err != nil {
		return nil, err
	}

	c, err := repo.CommitObject(hash)
	if err != nil {
		return nil, err
	}

	obj, err := c.File(path)
	if err != nil {
		if err == object.ErrFileNotFound {
			return nil, errFileDoesNotExist
		}
		return nil, err
	}

	contents, err := obj.Contents()
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Contents: contents,
		//RepoHash: ref.Hash().String(),
		ItemHash: obj.Hash.String(),
	}, nil
}

// GetFileNames retrieves a list of files names for a given path
func GetFileNames(
	db *proxy.DB,
	name string,
	branch string,
	path string) ([]string, error) {

	path = strings.Trim(path, "/")

	repo, err := openRepo(db, false, name)
	if err != nil {
		return nil, err
	}

	tree, _, err := resolveTreeFromName(repo, branch)

	if err != nil {
		return nil, err
	}

	if len(path) > 0 {
		tree, err = tree.Tree(path)
		if err != nil {
			if err == object.ErrDirectoryNotFound {
				return nil, errDoesNotExist
			}
			return nil, err
		}
	}

	var result []string
	for _, entry := range tree.Entries {
		if entry.Mode == filemode.Dir {
			result = append(result, entry.Name+"/")
		} else {
			result = append(result, entry.Name)
		}

	}

	return result, nil
}

// FilesTar represents a set of files to export
type FilesTar struct {
	File        *bytes.Buffer
	RepoHash    string
	ItemHashes  []string
	NotModified bool
}

// GetFiles retrieves multiple files in a .tar format
func GetFiles(
	db *proxy.DB,
	name string,
	branch string,
	path string,
	matchHash string) (*FilesTar, error) {

	rootPath := strings.TrimSuffix(strings.TrimSuffix(path, "*"), "/")

	repo, err := openRepo(db, false, name)
	if err != nil {
		return nil, err
	}

	tree, rh, err := resolveTreeFromName(repo, branch)
	if err != nil {
		return nil, err
	}

	repoHash := rh.String()

	if len(matchHash) > 0 && repoHash == matchHash {
		return &FilesTar{
			RepoHash:    repoHash,
			NotModified: true,
		}, nil
	}

	if len(rootPath) > 0 {
		tree, err = tree.Tree(rootPath)
		if err != nil {
			if err == object.ErrDirectoryNotFound {
				return &FilesTar{
					RepoHash: repoHash,
				}, nil
			}
			return nil, err
		}
	}

	var buf bytes.Buffer
	var itemHashes []string
	result := tar.NewWriter(&buf)
	fileIter := tree.Files()
	defer fileIter.Close()
	err = fileIter.ForEach(func(f *object.File) error {

		contents, err := f.Contents()
		if err != nil {
			return err
		}

		hdr := &tar.Header{
			Name: f.Name,
			Mode: 0600,
			Size: f.Size,
		}
		err = result.WriteHeader(hdr)
		if err != nil {
			return err
		}

		itemHashes = append(itemHashes, f.Hash.String())

		if _, err := result.Write([]byte(contents)); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	err = result.Close()
	if err != nil {
		return nil, err
	}

	return &FilesTar{
		File:        &buf,
		RepoHash:    repoHash,
		ItemHashes:  itemHashes,
		NotModified: false,
	}, nil
}

// UpdateFile commits the specified file back to the specified repository
func UpdateFile(
	db *proxy.DB,
	name string,
	path string,
	message string,
	content string,
	author string,
	email string,
	itemHash string,
	repoHash string,
	branch string) (*FileInfo, error) {

	path = strings.TrimPrefix(path, "/")

	repo, err := openRepo(db, true, name)

	if err != nil {
		return nil, err
	}

	err = compareHash(repo, path, itemHash, repoHash)
	if err != nil {
		return nil, err
	}

	w, err := repo.Worktree()

	if err != nil {
		return nil, err
	}

	f, err := w.Filesystem.Open(path)
	if err != nil {
		f, err = w.Filesystem.Create(path)
		if err != nil {
			return nil, err
		}
	}

	_, err = f.Write([]byte(content))
	if err != nil {
		return nil, err
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	_, err = w.Add(path)
	if err != nil {
		return nil, err
	}

	c, err := commitChanges(repo, w, message, author, email)
	if err != nil {
		return nil, err
	}

	// get the file
	obj, err := c.File(path)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		RepoHash: c.Hash.String(),
		ItemHash: obj.Hash.String(),
	}, nil
}

// DeleteFile removes the file specified from the specified repository
func DeleteFile(
	db *proxy.DB,
	name string,
	path string,
	message string,
	author string,
	email string,
	itemHash string,
	repoHash string,
	branch string) (*FileInfo, error) {

	path = strings.TrimPrefix(path, "/")

	repo, err := openRepo(db, true, name)

	if err != nil {
		return nil, err
	}

	err = compareHash(repo, path, itemHash, repoHash)
	if err != nil {
		return nil, err
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	_, err = w.Remove(path)
	if err != nil {
		return nil, err
	}

	c, err := commitChanges(repo, w, message, author, email)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		RepoHash: c.Hash.String(),
	}, nil
}

// CreateBranch creates a branch based on the specified existing branch
func CreateBranch(
	db *proxy.DB,
	name string,
	sourceBranch string,
	newBranch string) error {

	repo, err := openRepo(db, false, name)
	if err != nil {
		return err
	}

	hash, err := resolveHashFromName(repo, sourceBranch)
	if err != nil {
		return err
	}

	ref := plumbing.NewHashReference(refName(newBranch), hash)
	return repo.Storer.SetReference(ref)
}

// DeleteBranch creates a branch based on the specified existing branch
func DeleteBranch(
	db *proxy.DB,
	name string,
	branch string) error {

	s, err := sqlstore.NewStorage(db, name)
	if err != nil {
		return errDoesNotExist
	}

	return s.RemoveReference(refName(branch))
}

// DiffFile returns a diff between the requested versions
func DiffFile(
	db *proxy.DB,
	name string,
	path string,
	from string,
	to string) (string, error) {

	path = strings.TrimPrefix(path, "/")

	repo, err := openRepo(db, false, name)
	if err != nil {
		return "", err
	}

	result := bytes.NewBuffer(nil)
	encoder := diff.NewUnifiedEncoder(result, 1)

	fromTree, _, err := resolveTreeFromName(repo, from)
	if err != nil {
		return "", errInvalidReference
	}

	toTree, _, err := resolveTreeFromName(repo, to)
	if err != nil {
		return "", errInvalidReference
	}

	changes, err := fromTree.Diff(toTree)
	if err != nil {
		return "", err
	}

	for _, c := range changes {
		if c.To.Name == path {
			patch, err := c.Patch()
			if err != nil {
				return "", err
			}
			err = encoder.Encode(patch)
			if err != nil {
				return "", err
			}
			break
		}
	}

	return result.String(), nil
}

// Converts the branch name provided into a reference name
func refName(n string) plumbing.ReferenceName {
	if strings.HasPrefix(n, "refs/heads/") {
		return plumbing.ReferenceName(n)
	}
	return plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", n))
}

// resolve the name provided to a hash
func resolveHashFromName(repo *git.Repository, name string) (plumbing.Hash, error) {
	if len(name) == 0 {
		// default is HEAD
		head, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, err
		}
		return head.Hash(), nil
	}

	// Check if this is a reference
	ref, err := repo.Reference(refName(name), true)
	if err == nil && ref.Type() == plumbing.HashReference {
		return ref.Hash(), nil
	}
	if err != nil {
		return plumbing.ZeroHash, err
	}

	// check for a branch
	branch, err := repo.Branch(name)
	if err != nil && err != git.ErrBranchNotFound {
		// branch not found is ok, error for anything else
		return plumbing.ZeroHash, err
	}
	if err == nil {
		// return the branch
		reference, err := repo.Reference(branch.Merge, true)
		if err == nil && reference.Type() == plumbing.HashReference {
			return reference.Hash(), nil
		}
		return plumbing.NewHash(name), nil
	}

	// check for a tag
	tag, err := repo.Tag(name)
	if err == nil {
		return tag.Hash(), nil
	}

	fmt.Printf("NOT FOUND -- %s", name)

	return plumbing.NewHash(name), nil
}

func resolveTreeFromName(repo *git.Repository, name string) (*object.Tree, plumbing.Hash, error) {

	hash, err := resolveHashFromName(repo, name)
	if err != nil {
		return nil, hash, err
	}

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, plumbing.ZeroHash, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, plumbing.ZeroHash, err
	}

	return tree, commit.Hash, nil
}

// compareHash compares the given repository to the provided hashes, if hashes
// were provided and they do not match either the repository or the file (item)
// an error will be returned.
func compareHash(repo *git.Repository, path string, itemHash string, repoHash string) error {

	if len(itemHash) > 0 || len(repoHash) > 0 {
		head, err := repo.Head()
		if err != nil {
			return err
		}

		ref, err := repo.Reference(head.Name(), true)
		if err != nil {
			return err
		}
		c, err := repo.CommitObject(ref.Hash())
		if err != nil {
			return err
		}

		if len(repoHash) > 0 && c.Hash.String() != repoHash {
			return errHashConflict
		}

		if len(itemHash) > 0 {

			f, err := c.File(path)
			if err != nil {
				return err
			}

			if f.Hash.String() != itemHash {
				return errHashConflict
			}
		}
	}

	return nil
}

func commitChanges(
	repo *git.Repository,
	w *git.Worktree,
	message string,
	author string,
	email string) (*object.Commit, error) {

	h, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  author,
			Email: email,
			When:  time.Now(),
		},
	})

	if err != nil {
		return nil, err
	}

	master := plumbing.NewHashReference(plumbing.Master, h)
	err = repo.Storer.SetReference(master)

	if err != nil {
		return nil, err
	}

	c, err := repo.CommitObject(h)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func openRepo(store *proxy.DB, filesystem bool, repository string) (*git.Repository, error) {
	s, err := sqlstore.NewStorage(store, repository)

	if err != nil {
		return nil, errDoesNotExist
		//return nil, err
	}

	var fs billy.Filesystem

	if filesystem {
		fs = memfs.New()
	}

	return git.Open(s, fs)
}

func createRepo(
	db *proxy.DB,
	repository string) (*git.Repository, error) {

	s, err := sqlstore.NewStorage(db, repository)

	if err != nil {
		return nil, err
	}

	r, err := git.Init(s, nil)

	if err != nil {
		return nil, err
	}

	err = r.CreateBranch(&config.Branch{
		Name:   "master",
		Remote: "origin",
		Merge:  "refs/heads/master",
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}
