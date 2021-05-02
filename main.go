package main

import (
	"log"
	"time"

	"github.com/paulhatch/konfigraf/proxy"
	"github.com/paulhatch/konfigraf/service"

	"github.com/paulhatch/plgo"
)

// Creates a new repository and returns the repository ID.
func CreateRepository(repoName string) int {

	logger := plgo.NewNoticeLogger("konfigraf: ", log.Ltime)
	require(logger, repoName, "Repository name")

	db, err := plgo.Open()
	if err != nil {
		logger.Fatalf("Cannot open DB: %s", err)
	}
	defer db.Close()
	database := newProxy(db)

	id, err := service.CreateRepository(database, repoName)

	if err != nil {
		logger.Fatalf("Error: %s", err)
	}

	return id
}

// Deletes the specified repository.
func DeleteRepository(repoName string) {

	logger := plgo.NewNoticeLogger("konfigraf: ", log.Ltime)
	require(logger, repoName, "Repository name")

	db, err := plgo.Open()
	if err != nil {
		logger.Fatalf("Cannot open DB: %s", err)
	}
	defer db.Close()
	database := newProxy(db)

	err = service.DeleteRepository(database, repoName)

	if err != nil {
		logger.Fatalf("Error: %s", err)
	}
}

// Creates a new branch
func CreateBranch(repoName string, source string, newBranch string) {
	logger := plgo.NewNoticeLogger("konfigraf: ", log.Ltime)
	require(logger, repoName, "Repository name")
	require(logger, source, "Source")
	require(logger, newBranch, "New branch name")

	db, err := plgo.Open()
	if err != nil {
		logger.Fatalf("Cannot open DB: %s", err)
	}
	defer db.Close()
	database := newProxy(db)

	err = service.CreateBranch(database, repoName, source, newBranch)

	if err != nil {
		logger.Fatalf("Error: %s", err)
	}
}

// Removes an existing branch
func DeleteBranch(repoName string, branch string) {
	logger := plgo.NewNoticeLogger("konfigraf: ", log.Ltime)
	require(logger, repoName, "Repository name")
	require(logger, branch, "Branch name")

	db, err := plgo.Open()
	if err != nil {
		logger.Fatalf("Cannot open DB: %s", err)
	}
	defer db.Close()
	database := newProxy(db)

	err = service.DeleteBranch(database, repoName, branch)
	if err != nil {
		logger.Fatalf("Error: %s", err)
	}
}

// Commits a file on the master branch
func CommitFile(repoName string, path string, content string, author string, message string, email string) []string {
	return commitFileImpl(repoName, path, content, author, message, email, "master")
}

// Commits a file on a specific branch
func CommitBranchFile(repoName string, branch string, path string, content string, author string, message string, email string) []string {
	return commitFileImpl(repoName, path, content, author, message, email, branch)
}

// Shared implementations need to be unexported since PLGO will change the
// return type to Datum meaning we cannot call any exported method in the main
// package internally.

func commitFileImpl(repoName string, path string, content string, author string, message string, email string, branch string) []string {
	logger := plgo.NewNoticeLogger("konfigraf: ", log.Ltime)
	require(logger, repoName, "Repository name")
	require(logger, branch, "Repository name")
	require(logger, path, "Path")

	db, err := plgo.Open()
	if err != nil {
		logger.Fatalf("Cannot open DB: %s", err)
	}
	defer db.Close()
	database := newProxy(db)

	info, err := service.UpdateFile(database, repoName, path, message, content, author, email, "", "", branch)

	if err != nil {
		logger.Fatalf("Error: %s", err)
	}

	return []string{info.ItemHash, info.RepoHash}
}

// Gets a file from a repository at the specified path of the master branch.
func GetFile(repoName string, path string) string {
	return getFileImpl(repoName, path, "master")
}

// Gets a file from a repository at the specified path of the master branch.
func GetBranchFile(repoName string, branch string, path string) string {
	return getFileImpl(repoName, path, branch)
}

func getFileImpl(repoName string, path string, branch string) string {
	logger := plgo.NewNoticeLogger("konfigraf: ", log.Ltime)
	require(logger, repoName, "Repository name")
	require(logger, path, "Path")
	require(logger, branch, "Branch name")

	db, err := plgo.Open()
	if err != nil {
		logger.Fatalf("Cannot open DB: %s", err)
	}
	defer db.Close()
	database := newProxy(db)

	info, err := service.GetFile(database, repoName, branch, path)

	if err != nil {
		logger.Fatalf("Error: %s", err)
	}

	return info.Contents
}

// Lists files from a specific path of the master branch
func ListFiles(repoName string, path string) []string {
	return listFilesImpl(repoName, path, "master")
}

// Lists files from a specific path of the specified branch
func ListBranchFiles(repoName string, branch string, path string) []string {
	return listFilesImpl(repoName, path, branch)
}

func listFilesImpl(repoName string, path string, branch string) []string {
	logger := plgo.NewNoticeLogger("konfigraf: ", log.Ltime)
	require(logger, repoName, "Repository name")
	require(logger, branch, "Branch name")

	db, err := plgo.Open()
	if err != nil {
		logger.Fatalf("Cannot open DB: %s", err)
	}
	defer db.Close()
	database := newProxy(db)

	files, err := service.GetFileNames(database, repoName, branch, path)

	if err != nil {
		logger.Fatalf("Error: %s", err)
	}

	return files
}

// Gets log from the master branch
func GetLog(repoName string, since *time.Time, until *time.Time, file string) []string {
	return getHistoryImpl(repoName, "master", since, until, file)
}

// Lists files from a specific path of the master branch
func GetBranchLog(repoName string, branch string, since *time.Time, until *time.Time, file string) []string {
	return getHistoryImpl(repoName, branch, since, until, file)
}

func getHistoryImpl(repoName string, branch string, since *time.Time, until *time.Time, file string) []string {
	logger := plgo.NewNoticeLogger("konfigraf: ", log.Ltime)
	require(logger, repoName, "Repository name")
	require(logger, branch, "Branch name")

	db, err := plgo.Open()
	if err != nil {
		logger.Fatalf("Cannot open DB: %s", err)
	}
	defer db.Close()
	database := newProxy(db)

	history, err := service.GetHistory(database, repoName, branch, since, until, nil)

	if err != nil {
		logger.Fatalf("Error: %s", err)
	}

	return history
}

// Validation method for strings
func require(l *log.Logger, v string, n string) {
	if len(v) == 0 {
		l.Fatalf("%s is required", n)
	}
}

// Create a proxy API to access the database with
func newProxy(db *plgo.DB) *proxy.DB {

	exec := func(query string, types []string, args []interface{}) error {
		stmt, err := db.Prepare(query, types)
		if err != nil {
			return err
		}

		return stmt.Exec(args...)
	}

	query := func(query string, types []string, args []interface{}) (*proxy.Rows, error) {
		stmt, err := db.Prepare(query, types)
		if err != nil {
			return nil, err
		}

		rows, err := stmt.Query(args...)
		if err != nil {
			return nil, err
		}

		next := func() bool {
			return rows.Next()
		}

		scan := func(args []interface{}) error {
			return rows.Scan(args...)
		}

		return &proxy.Rows{next, scan}, nil
	}

	queryRow := func(query string, types []string, args []interface{}) (*proxy.Row, error) {
		stmt, err := db.Prepare(query, types)
		if err != nil {
			return nil, err
		}

		row, err := stmt.QueryRow(args...)
		if err != nil {
			return nil, err
		}

		scan := func(args []interface{}) error {
			return row.Scan(args...)
		}

		return &proxy.Row{scan}, nil
	}

	return &proxy.DB{exec, query, queryRow}
}
